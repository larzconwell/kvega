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

// InvalidRowError represents invalid database row data and the cause for the invalid data.
type InvalidRowError struct {
	row   int
	cause error
}

// Error implements error interface.
func (ire *InvalidRowError) Error() string {
	return fmt.Sprintf("invalid row data on row %d: %s", ire.row, ire.cause.Error())
}

// Unwrap implements error unwrapping for errors.Is/errors.As.
func (ire *InvalidRowError) Unwrap() error {
	return ire.cause
}

// EmbeddedDB is an implementation of DB that provides access to a database backed by
// a file stored on the local disk.
type EmbeddedDB struct {
	path   string
	index  map[string]int64
	closed atomic.Bool

	mu   sync.Mutex
	file *os.File
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

	index, err := buildIndex(file)
	if err != nil {
		// The index building error is more important to surface.
		//nolint:errcheck
		//gosec:disable G104
		file.Close()

		return nil, err
	}

	return &EmbeddedDB{path: path, index: index, file: file}, nil
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

	encodedKeyLen := base64.StdEncoding.EncodedLen(len(key))
	encodedValueLen := base64.StdEncoding.EncodedLen(len(value))
	row := make([]byte, 4+encodedKeyLen+encodedValueLen)

	row[0] = setType
	row[1] = ','
	base64.StdEncoding.Encode(row[2:], []byte(key))
	row[2+encodedKeyLen] = ','
	base64.StdEncoding.Encode(row[3+encodedKeyLen:], value)
	row[3+encodedKeyLen+encodedValueLen] = '\n'

	edb.mu.Lock()
	defer edb.mu.Unlock()

	_, err := edb.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	_, err = edb.file.Write(row)
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	return nil
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

	_, err := edb.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	var (
		count int
		value []byte
	)

	reader := csv.NewReader(edb.file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	for {
		count++

		columns, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		switch columns[0] {
		case string(setType):
			if len(columns) != 3 {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumns}
			}

			encodedKey := columns[1]
			encodedValue := columns[2]

			if len(encodedKey) == 0 {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
			}

			decodedKey, err := base64.StdEncoding.DecodeString(encodedKey)
			if err != nil {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
			}

			if string(decodedKey) == key {
				decodedValue, err := base64.StdEncoding.DecodeString(encodedValue)
				if err != nil {
					return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
				}

				value = decodedValue
			}
		case string(deleteType):
			if len(columns) != 2 {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumns}
			}

			encodedKey := columns[1]
			if len(encodedKey) == 0 {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
			}

			decodedKey, err := base64.StdEncoding.DecodeString(encodedKey)
			if err != nil {
				return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
			}

			if string(decodedKey) == key {
				value = nil
			}
		default:
			return nil, &InvalidRowError{row: count, cause: ErrInvalidRowType}
		}
	}

	if value == nil {
		return nil, ErrNotFound
	}

	return value, nil
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

	encodedKeyLen := base64.StdEncoding.EncodedLen(len(key))
	row := make([]byte, 3+encodedKeyLen)

	row[0] = deleteType
	row[1] = ','
	base64.StdEncoding.Encode(row[2:], []byte(key))
	row[2+encodedKeyLen] = '\n'

	edb.mu.Lock()
	defer edb.mu.Unlock()

	_, err := edb.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	_, err = edb.file.Write(row)
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	return nil
}

// buildIndex builds an index of keys to their respective offset in the file,
// it locates the last instance of any keys found in the file. The file is
// expected to be set to the beginning of the file, i.e. no seek occurs.
// buildIndex only does minimal validation to get valid keys from the file.
func buildIndex(file *os.File) (map[string]int64, error) {
	var (
		count  int
		offset int64
	)

	index := make(map[string]int64)

	reader := csv.NewReader(file)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	for {
		count++

		columns, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		// All row type have at least 2 columns, the row type, and a key.
		if len(columns) < 2 {
			return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumns}
		}

		decodedKey, err := base64.StdEncoding.DecodeString(columns[1])
		if err != nil {
			return nil, &InvalidRowError{row: count, cause: ErrInvalidRowColumn}
		}

		index[string(decodedKey)] = offset
		offset = reader.InputOffset()
	}

	return index, nil
}
