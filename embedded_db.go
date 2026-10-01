package kvega

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/larzconwell/kvega/binfmt"
)

var (
	// ErrClosed is returned when a database has been closed.
	ErrClosed = errors.New("kvega: database closed")
	// ErrNotFound is returned when a key could not be found.
	ErrNotFound = errors.New("kvega: key not found")
	// ErrEmptyKey is returned when the given key is empty.
	ErrEmptyKey = errors.New("kvega: empty key")
)

// EmbeddedDB is an implementation of DB that provides access to a database backed by
// a file stored on the local disk.
type EmbeddedDB struct {
	path   string
	closed atomic.Bool

	imu   sync.Mutex
	index map[string]int64

	wmu    sync.Mutex
	writer *os.File

	readerCounter atomic.Int32
	rmus          []sync.Mutex
	readers       []*os.File
}

// OpenEmbeddedDB opens the database located at the given path, creating it if it doesn't exist.
// All writes to the database are flushed to disk at the time of the write.
//
// The returned [EmbeddedDB] is safe for concurrent use by multiple goroutines, and as a result
// should only require one call.
//
// The returned [EmbeddedDB] should be closed when finished to ensure caches are flushed.
func OpenEmbeddedDB(path string) (*EmbeddedDB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0o750)
	if err != nil {
		return nil, fmt.Errorf("kvega: failed to create directories: %w", err)
	}

	// The caller should ensure the provided path is safe to open.
	//gosec:disable G304
	writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("kvega: failed to create writer: %w", err)
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

			return nil, fmt.Errorf("kvega: failed to create reader: %w", err)
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

// Close handles flushing caches and releasing resources related to the [EmbeddedDB].
//
// [ErrClosed] is returned if the database has been closed.
func (edb *EmbeddedDB) Close() error {
	if edb.closed.Load() {
		return ErrClosed
	}

	edb.closed.Store(true)

	edb.wmu.Lock()
	defer edb.wmu.Unlock()

	err := edb.writer.Sync()
	if err != nil {
		return fmt.Errorf("kvega: failed to sync writer: %w", err)
	}

	err = edb.writer.Close()
	if err != nil {
		return fmt.Errorf("kvega: failed to close writer: %w", err)
	}

	for idx, reader := range edb.readers {
		edb.rmus[idx].Lock()
		err := reader.Close()
		edb.rmus[idx].Unlock()

		if err != nil {
			return fmt.Errorf("kvega: failed to close reader: %w", err)
		}
	}

	return nil
}

// Set handles setting the key to the provided value.
//
// [ErrClosed] is returned if the database has been closed.
func (edb *EmbeddedDB) Set[VT ValueType[T], T any](key string, value T) error {
	if edb.closed.Load() {
		return ErrClosed
	}

	if key == "" {
		return ErrEmptyKey
	}

	return edb.encodeRow(func(enc *binfmt.Encoder) (string, error) {
		return key, enc.SetRow[VT](key, value)
	})
}

// Delete handles deleting the provided key if one exists.
//
// [ErrClosed] is returned if the database has been closed.
func (edb *EmbeddedDB) Delete(key string) error {
	if edb.closed.Load() {
		return ErrClosed
	}

	if key == "" {
		return ErrEmptyKey
	}

	return edb.encodeRow(func(enc *binfmt.Encoder) (string, error) {
		return key, enc.DeleteRow(key)
	})
}

// Get returns the value that's associated with the key if one exists.
//
// [ErrNotFound] is returned if the key was not found.
// [ErrClosed] is returned if the database has been closed.
func (edb *EmbeddedDB) Get[VT ValueType[T], T any](key string) (T, error) {
	var zeroT T

	if edb.closed.Load() {
		return zeroT, ErrClosed
	}

	if key == "" {
		return zeroT, ErrEmptyKey
	}

	idx := int(edb.readerCounter.Add(1)) % len(edb.readers)
	idx = max(-idx, idx) // Non branching way to get absolute value, doesn't work at math.MinInt

	edb.rmus[idx].Lock()
	defer edb.rmus[idx].Unlock()

	edb.imu.Lock()
	offset, foundOffset := edb.index[key]
	edb.imu.Unlock()

	file := edb.readers[idx]

	_, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return zeroT, fmt.Errorf("kvega: failed to seek reader: %w", err)
	}

	var (
		found bool
		value T
	)

	decoder := binfmt.NewDecoder(file)

	for {
		_, rowKey, rowValueDecoder, _, err := edb.decodeRow(decoder)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return zeroT, err
		}

		if rowKey == key {
			if rowValueDecoder.HasValue {
				rowValue, _, err := rowValueDecoder.Get[VT]()
				if err != nil {
					return zeroT, fmt.Errorf("kvega: failed to decode value: %w", err)
				}

				found = true
				value = rowValue
			} else {
				found = false // If decode has no value then it's a delete row.
			}
		} else if rowValueDecoder.HasValue {
			_, err := rowValueDecoder.Skip()
			if err != nil {
				return zeroT, fmt.Errorf("kvega: failed to decode value: %w", err)
			}
		}

		// Only decode a single row if we found offset in index.
		if foundOffset {
			break
		}
	}

	if !found {
		return zeroT, ErrNotFound
	}

	return value, nil
}

// encodeRow encodes row data by filling a buffer using the given
// fill function and updates the index for the returned key to
// the offset to the newly encoded row.
func (edb *EmbeddedDB) encodeRow(encode func(enc *binfmt.Encoder) (string, error)) error {
	edb.wmu.Lock()
	defer edb.wmu.Unlock()

	// Doesn't actually seek, retrieves current offset.
	offset, err := edb.writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("kvega: failed to seek writer: %w", err)
	}

	enc := binfmt.NewEncoder()

	key, err := encode(enc)
	if err != nil {
		return fmt.Errorf("kvega: failed to encode row: %w", err)
	}

	_, err = enc.WriteTo(edb.writer)
	if err != nil {
		return fmt.Errorf("kvega: failed to write row: %w", err)
	}

	edb.imu.Lock()
	edb.index[key] = offset
	edb.imu.Unlock()

	return nil
}

// decodeRow decodes one row from the reader, returning
// io.EOF if the end of the file has been reached.
func (edb *EmbeddedDB) decodeRow(dec *binfmt.Decoder) (byte, string, binfmt.RowValueDecoder, int, error) {
	ident, key, valueDecoder, n, err := dec.Row()
	if err != nil {
		return ident, key, valueDecoder, n, fmt.Errorf("kvega: failed to decode row: %w", err)
	}

	return ident, key, valueDecoder, n, nil
}

// buildIndex builds an index of keys to their respective offset in the file,
// it locates the last instance of any keys found in the file. buildIndex
// does not take a lock, it is not safe to call concurrently. It only does
// minimal validation to get valid keys from the file.
func (edb *EmbeddedDB) buildIndex() error {
	file := edb.readers[0]

	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("kvega: failed to seek reader: %w", err)
	}

	var offset int64

	index := make(map[string]int64)
	decoder := binfmt.NewDecoder(file)

	for {
		_, key, valueDecoder, rown, err := edb.decodeRow(decoder)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		var valuen int
		if valueDecoder.HasValue {
			valuen, err = valueDecoder.Skip()
			if err != nil {
				return fmt.Errorf("kvega: failed to decode value: %w", err)
			}
		}

		index[key] = offset
		offset += int64(rown) + int64(valuen)
	}

	edb.index = index

	return nil
}
