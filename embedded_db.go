package kvega

import (
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
)

var _ DB = (*EmbeddedDB)(nil)

const (
	setType    = 'S'
	deleteType = 'D'
)

var (
	// ErrEmptyKey is returned when the given key is empty.
	ErrEmptyKey = errors.New("empty key")
	// ErrInvalidRowType is returned when row data has an invalid type.
	ErrInvalidRowType = errors.New("invalid row type")
	// ErrInvalidRowColumns is returned when row data contains an invalid number of columns.
	ErrInvalidRowColumns = errors.New("invalid row columns")
	// ErrInvalidRowColumn is returned when row data contains invalid column data.
	ErrInvalidRowColumn = errors.New("invalid row column")
)

type row struct {
	typ          byte
	key          string
	encodedValue *string
}

// InvalidRowError represents invalid database row data and the cause for the invalid data.
type InvalidRowError struct {
	key   string
	cause error
}

// Error implements error interface.
func (ire *InvalidRowError) Error() string {
	if ire.key != "" {
		return fmt.Sprintf(`invalid row data for key "%s": %s`, ire.key, ire.cause.Error())
	}

	return "invalid row data: " + ire.cause.Error()
}

// Unwrap implements error unwrapping for errors.Is/errors.As.
func (ire *InvalidRowError) Unwrap() error {
	return ire.cause
}

// EmbeddedDB is an implementation of DB that provides access to a database backed by
// a file stored on the local disk.
type EmbeddedDB struct {
	path   string
	closed atomic.Bool

	imu   sync.Mutex
	index map[string]int64

	wmu    sync.Mutex
	writer *os.File

	readCounter atomic.Int32
	rmus        []sync.Mutex
	readers     []*os.File
}

// OpenEmbeddedDB opens the database located at the given path, creating it if it doesn't exist.
// All writes to the database are flushed to disk at the time of the write.
//
// The returned EmbeddedDB is safe for concurrent use by multiple goroutines, and as a result
// should only require one call.
//
// The returned EmbeddedDB should be closed when finished to ensure caches are flushed.
func OpenEmbeddedDB(path string) (*EmbeddedDB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	// The caller should ensure the provided path is safe to open.
	//gosec:disable G304
	writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}

	readers := make([]*os.File, runtime.GOMAXPROCS(0))
	for idx := range readers {
		// The caller should ensure the provided path is safe to open.
		//gosec:disable G304
		reader, err := os.Open(path)
		if err != nil {
			// The last open error is more important to surface.
			//nolint:errcheck
			//gosec:disable G104
			writer.Close()

			for _, reader := range readers[:idx] {
				// The last open error is more important to surface.
				//nolint:errcheck
				//gosec:disable G104
				reader.Close()
			}

			return nil, fmt.Errorf("failed to create reader: %w", err)
		}

		readers[idx] = reader
	}

	edb := &EmbeddedDB{
		path:    path,
		writer:  writer,
		rmus:    make([]sync.Mutex, len(readers)),
		readers: readers,
	}

	err = edb.buildIndex()
	if err != nil {
		// The index building error is more important to surface.
		//nolint:errcheck
		//gosec:disable G104
		writer.Close()

		for _, reader := range readers {
			// The index building error is more important to surface.
			//nolint:errcheck
			//gosec:disable G104
			reader.Close()
		}

		return nil, err
	}

	return edb, nil
}

// Close handles flushing caches and releasing resources related to the EmbeddedDB.
//
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Close() error {
	if edb.closed.Load() {
		return ErrClosed
	}

	edb.closed.Store(true)

	edb.wmu.Lock()
	defer edb.wmu.Unlock()

	err := edb.writer.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync writer: %w", err)
	}

	err = edb.writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	for idx, reader := range edb.readers {
		edb.rmus[idx].Lock()
		err := reader.Close()
		edb.rmus[idx].Unlock()

		if err != nil {
			return fmt.Errorf("failed to close reader: %w", err)
		}
	}

	return nil
}

// Set handles setting the key to the provided value.
//
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Set(key string, value []byte) error {
	if edb.closed.Load() {
		return ErrClosed
	}

	if key == "" {
		return ErrEmptyKey
	}

	// Value needs to be allocated so it writes a value to the file.
	if value == nil {
		value = make([]byte, 0)
	}

	return edb.writeRow(setType, []byte(key), value)
}

// Delete handles deleting the provided key if one exists.
//
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Delete(key string) error {
	if edb.closed.Load() {
		return ErrClosed
	}

	if key == "" {
		return ErrEmptyKey
	}

	return edb.writeRow(deleteType, []byte(key), nil)
}

// Get returns the value that's associated with the key if one exists.
//
// ErrNotFound is returned if the key was not found.
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Get(key string) ([]byte, error) {
	if edb.closed.Load() {
		return nil, ErrClosed
	}

	if key == "" {
		return nil, ErrEmptyKey
	}

	idx := int(edb.readCounter.Add(1)) % len(edb.readers)
	idx = max(-idx, idx) // Non branching way to get absolute value, doesn't work at math.MinInt

	edb.rmus[idx].Lock()
	defer edb.rmus[idx].Unlock()

	edb.imu.Lock()
	offset, foundOffset := edb.index[key]
	edb.imu.Unlock()

	file := edb.readers[idx]

	_, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek reader: %w", err)
	}

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	var foundRow row

	if foundOffset {
		row, err := edb.readRow(reader)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		if row.key == key {
			foundRow = row
		}
	} else {
		for {
			row, err := edb.readRow(reader)
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				return nil, err
			}

			if row.key == key {
				foundRow = row
			}
		}
	}

	if foundRow.key == "" {
		return nil, ErrNotFound
	}

	switch foundRow.typ {
	case setType:
		if foundRow.encodedValue == nil {
			return nil, &InvalidRowError{key: key, cause: ErrInvalidRowColumns}
		}

		decodedValue, err := base64.StdEncoding.DecodeString(*foundRow.encodedValue)
		if err != nil {
			return nil, &InvalidRowError{key: key, cause: ErrInvalidRowColumn}
		}

		return decodedValue, nil
	case deleteType:
		if foundRow.encodedValue != nil {
			return nil, &InvalidRowError{key: key, cause: ErrInvalidRowColumns}
		}

		return nil, ErrNotFound
	default:
		return nil, &InvalidRowError{key: key, cause: ErrInvalidRowType}
	}
}

// writeRow writes the given row data and updates the index for the
// key to point to the offset for the newly written row. The value
// argument is only written if not nil, care must be taken by the
// caller to ensure that nil is only passed if there is no value
// intended to be written.
func (edb *EmbeddedDB) writeRow(typ byte, key, value []byte) error {
	var encodedValueLen int

	baseRowLen := 3
	encodedKeyLen := base64.StdEncoding.EncodedLen(len(key))

	if value != nil {
		baseRowLen++
		encodedValueLen = base64.StdEncoding.EncodedLen(len(value))
	}

	row := make([]byte, baseRowLen+encodedKeyLen+encodedValueLen)

	row[0] = typ
	row[1] = ','
	rowOffset := 2

	base64.StdEncoding.Encode(row[rowOffset:], key)
	rowOffset += encodedKeyLen

	if value != nil {
		row[rowOffset] = ','
		rowOffset++

		base64.StdEncoding.Encode(row[rowOffset:], value)
		rowOffset += encodedValueLen
	}

	row[rowOffset] = '\n'

	edb.wmu.Lock()
	defer edb.wmu.Unlock()

	// Doesn't actually seek, retrieves current offset.
	offset, err := edb.writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("failed to seek writer: %w", err)
	}

	_, err = edb.writer.Write(row)
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	edb.imu.Lock()
	edb.index[string(key)] = offset
	edb.imu.Unlock()

	return nil
}

// readRow reads one row from the given CSV reader, returning io.EOF
// if we've reached the end of the file. All validation are done in
// readRow except validating the row type and any value in the row.
func (edb *EmbeddedDB) readRow(reader *csv.Reader) (row, error) {
	columns, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return row{}, io.EOF
	}

	if err != nil {
		return row{}, fmt.Errorf("failed to read row: %w", err)
	}

	if len(columns) < 2 {
		return row{}, &InvalidRowError{cause: ErrInvalidRowColumns}
	}

	if len(columns[0]) != 1 {
		return row{}, &InvalidRowError{cause: ErrInvalidRowType}
	}

	typ := columns[0][0]

	encodedKey := columns[1]
	if len(encodedKey) == 0 {
		return row{}, &InvalidRowError{cause: ErrInvalidRowColumn}
	}

	decodedKey, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return row{}, &InvalidRowError{cause: ErrInvalidRowColumn}
	}

	var encodedValue *string
	if len(columns) > 2 {
		encodedValue = &columns[2]
	}

	return row{
		typ:          typ,
		key:          string(decodedKey),
		encodedValue: encodedValue,
	}, nil
}

// buildIndex builds an index of keys to their respective offset in the file,
// it locates the last instance of any keys found in the file. buildIndex
// does not take a lock, it is not safe to call concurrently. It only does
// minimal validation to get valid keys from the file.
func (edb *EmbeddedDB) buildIndex() error {
	var offset int64

	file := edb.readers[0]

	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek reader: %w", err)
	}

	index := make(map[string]int64)
	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	for {
		row, err := edb.readRow(reader)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		index[row.key] = offset
		offset = reader.InputOffset()
	}

	edb.index = index

	return nil
}
