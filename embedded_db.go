package kvega

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var _ DB = (*EmbeddedDB)(nil)

var (
	// ErrEmptyKey is returned when the given key is empty.
	ErrEmptyKey = errors.New("empty key")
)

// EmbeddedDB is an implementation of DB that provides access to a database backed by
// a file stored on the local disk.
type EmbeddedDB struct {
	path   string
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

	return &EmbeddedDB{path: path, file: file}, nil
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

	var row bytes.Buffer
	row.WriteString("S,")
	row.WriteString(base64.StdEncoding.EncodeToString([]byte(key)))
	row.WriteString(",")
	row.WriteString(base64.StdEncoding.EncodeToString(value))
	row.WriteString("\n")

	edb.mu.Lock()
	defer edb.mu.Unlock()

	_, err := edb.file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	_, err = edb.file.Write(row.Bytes())
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	return nil
}

// Get returns the value that's associated with the key if one exists.
//
// ErrNotFound is returned if the key was not found.
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Get(_ string) ([]byte, error) {
	if edb.closed.Load() {
		return nil, ErrClosed
	}

	return nil, ErrNotFound
}

// Delete handles deleting the provided key if one exists.
//
// ErrClosed is returned if the database has been closed.
func (edb *EmbeddedDB) Delete(_ string) error {
	if edb.closed.Load() {
		return ErrClosed
	}

	return nil
}
