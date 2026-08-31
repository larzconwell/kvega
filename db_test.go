package kvega

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenDB(t *testing.T) {
	t.Parallel()

	t.Run("returns error if invalid url", func(t *testing.T) {
		t.Parallel()

		db, err := OpenDB(":") // Force error by missing scheme
		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("returns error if invalid scheme", func(t *testing.T) {
		t.Parallel()

		db, err := OpenDB("gopher://host/9/db/test.kvega")
		assert.Nil(t, db)
		assert.ErrorIs(t, err, ErrInvalidLocationScheme)
	})

	t.Run("file scheme", func(t *testing.T) {
		t.Parallel()

		t.Run("returns error if host given", func(t *testing.T) {
			t.Parallel()

			location := &url.URL{
				Scheme: "file",
				Host:   "host",
				Path:   fsToURLPath(filepath.Join(t.ArtifactDir(), "db.kvega")),
			}

			db, err := OpenDB(location.String())
			assert.Nil(t, db)
			assert.ErrorIs(t, err, ErrInvalidEmbeddedDBLocation)
		})

		t.Run("returns error if no path given", func(t *testing.T) {
			t.Parallel()

			location := &url.URL{Scheme: "file"}

			db, err := OpenDB(location.String())
			assert.Nil(t, db)
			assert.ErrorIs(t, err, ErrInvalidEmbeddedDBLocation)
		})

		t.Run("returns error from OpenEmbeddedDB", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			// Force error by creating the path as a directory.
			err := os.MkdirAll(path, 0o750)
			assert.NoError(t, err)

			location := &url.URL{
				Scheme: "file",
				Path:   fsToURLPath(path),
			}

			db, err := OpenDB(location.String())
			assert.Nil(t, db)
			assert.Error(t, err)
		})

		t.Run("returns embedded db", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			location := &url.URL{
				Scheme: "file",
				Path:   fsToURLPath(path),
			}

			db, err := OpenDB(location.String())
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			edb, ok := db.(*EmbeddedDB)
			if !ok {
				assert.Fail(t, "Returned DB should be an *EmbeddedDB")
			}

			assert.Equal(t, path, edb.path)
		})
	})
}
