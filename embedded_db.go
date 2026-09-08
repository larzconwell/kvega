package kvega

import (
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	encodedValue []byte
}

// InvalidRowError represents invalid database row data and the cause for the invalid data.
type InvalidRowError struct {
	cause error
}

// Error implements error interface.
func (ire *InvalidRowError) Error() string {
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

	mu    sync.Mutex
	index map[string]int64
	file  *os.File
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
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	edb := &EmbeddedDB{path: path, file: file}

	err = edb.buildIndex()
	if err != nil {
		// The index building error is more important to surface.
		//nolint:errcheck
		//gosec:disable G104
		file.Close()

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

	edb.mu.Lock()
	defer edb.mu.Unlock()

	err := edb.file.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	err = edb.file.Close()
	if err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	edb.closed.Store(true)

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

	edb.mu.Lock()
	defer edb.mu.Unlock()

	offset, foundOffset := edb.index[key]

	_, err := edb.file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	reader := csv.NewReader(edb.file)
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
			return nil, &InvalidRowError{cause: ErrInvalidRowColumns}
		}

		decodedValue := make([]byte, base64.StdEncoding.EncodedLen(len(foundRow.encodedValue)))

		n, err := base64.StdEncoding.Decode(decodedValue, foundRow.encodedValue)
		if err != nil {
			return nil, &InvalidRowError{cause: ErrInvalidRowColumn}
		}

		return decodedValue[:n], nil
	case deleteType:
		if foundRow.encodedValue != nil {
			return nil, &InvalidRowError{cause: ErrInvalidRowColumns}
		}

		return nil, ErrNotFound
	default:
		return nil, &InvalidRowError{cause: ErrInvalidRowType}
	}
}

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

	edb.mu.Lock()
	defer edb.mu.Unlock()

	offset, err := edb.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	_, err = edb.file.Write(row)
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	edb.index[string(key)] = offset

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

	var encodedValue []byte
	if len(columns) > 2 {
		encodedValue = []byte(columns[2])
	}

	return row{
		typ:          typ,
		key:          string(decodedKey),
		encodedValue: encodedValue,
	}, nil
}

// buildIndex builds an index of keys to their respective offset in the file,
// it locates the last instance of any keys found in the file. buildIndex only
// does minimal validation to get valid keys from the file.
func (edb *EmbeddedDB) buildIndex() error {
	var offset int64

	_, err := edb.file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	index := make(map[string]int64)
	reader := csv.NewReader(edb.file)
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
