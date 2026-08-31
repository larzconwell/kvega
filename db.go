package kvega

import (
	"errors"
	"fmt"
	"io"
	"net/url"
)

var (
	// ErrInvalidLocationScheme is returned by OpenDB when an invalid scheme is given.
	ErrInvalidLocationScheme = errors.New("invalid location scheme")
	// ErrInvalidEmbeddedDBLocation is returned by OpenDB when the location is missing
	// the path or a host is given.
	ErrInvalidEmbeddedDBLocation = errors.New("invalid embedded db location")
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

// OpenDB opens a database at the given URL location, where the scheme dictates
// whether the database is embedded or being served by a kvega database server.
//
// To open an embedded database use the file scheme following the below examples:
// - file:///var/kvega/production.kvega
// - file:///C:/ProgramData/kvega/production.kvega
//
// The returned DB is safe for concurrent use by multiple goroutines, and as a result
// should only require one call.
//
// The returned DB should be closed when finished to ensure caches are flushed.
//
//nolint:ireturn
func OpenDB(location string) (DB, error) {
	url, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("failed to parse location: %w", err)
	}

	switch url.Scheme {
	case "file":
		if url.Host != "" {
			return nil, ErrInvalidEmbeddedDBLocation
		}

		if url.Path == "" {
			return nil, ErrInvalidEmbeddedDBLocation
		}

		return OpenEmbeddedDB(urlToFSPath(url.Path))
	default:
		return nil, ErrInvalidLocationScheme
	}
}
