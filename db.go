package kvega

import (
	"errors"
	"io"
)

var (
	// ErrClosed is returned when a database has been closed.
	ErrClosed = errors.New("database closed")
	// ErrNotFound is returned when a key could not be found.
	ErrNotFound = errors.New("key not found")
)

// A DB provides access to a database backend.
//
// Implementations of DB must be safe for concurrent use by multiple goroutines.
type DB interface {
	// Close handles flushing caches and releasing resources related to the EmbeddedDB.
	//
	// ErrClosed is returned if the database has been closed.
	io.Closer

	// Set handles setting the key to the provided value.
	//
	// ErrClosed is returned if the database has been closed.
	Set(key string, value Binary) error
	// Get returns the value that's associated with the key if one exists.
	//
	// ErrNotFound is returned if the key was not found.
	// ErrClosed is returned if the database has been closed.
	Get(key string) (Binary, error)
	// Delete handles deleting the provided key if one exists.
	//
	// ErrClosed is returned if the database has been closed.
	Delete(key string) error
}
