package kvega

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

var _ DB = (*EmbeddedDB)(nil)

// EmbeddedDB is an implementation of DB that provides access to a database backed by
// a file stored on the local disk.
type EmbeddedDB struct {
	path   string
	file   *os.File
	closed atomic.Bool
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
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o600)
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
func (edb *EmbeddedDB) Set(_ string, _ []byte) error {
	if edb.closed.Load() {
		return ErrClosed
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
