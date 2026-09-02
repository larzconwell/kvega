package kvega

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenEmbeddedDB(t *testing.T) {
	t.Parallel()

	t.Run("returns error if failed to create directory", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "file", "db.kvega")

		// Force error by creating the parent directory of the file as a file.
		file, err := os.Create(filepath.Dir(path))
		assert.NoError(t, err)
		assert.NoError(t, file.Close())

		db, err := OpenEmbeddedDB(path)
		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("returns error if failed to create file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		// Force error by creating the path as a directory.
		err := os.MkdirAll(path, 0o750)
		assert.NoError(t, err)

		db, err := OpenEmbeddedDB(path)
		assert.Nil(t, db)
		assert.Error(t, err)
	})

	t.Run("returns embedded db", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		// File used to confirm database appends new bytes to any existing bytes.
		//gosec:disable G304
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_SYNC, 0o600)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, file.Close())
		}()

		_, err = file.WriteString("1")
		assert.NoError(t, err)

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		assert.Equal(t, path, db.path)

		// Confirm that we opened in write only
		_, err = db.file.Read(make([]byte, 1))
		assert.Error(t, err)

		_, err = db.file.WriteString("2")
		assert.NoError(t, err)

		// Seek to beginning to confirm old and new data exist.
		_, err = file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		data := make([]byte, 2)
		_, err = file.Read(data)
		assert.NoError(t, err)
		assert.Equal(t, []byte("12"), data)
	})
}

func TestEmbeddedDBClose(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, db.Close())

		assert.ErrorIs(t, db.Close(), ErrClosed)
	})

	t.Run("returns error if failed to sync file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		// Force error by manually closing file.
		assert.NoError(t, db.file.Close())

		assert.Error(t, db.Close())
	})

	t.Run("sets closed state", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, db.Close())

		assert.True(t, db.closed.Load())
	})
}

func TestEmbeddedDBSet(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, db.Close())

		assert.ErrorIs(t, db.Set("key", []byte("value")), ErrClosed)
	})
}

func TestEmbeddedDBGet(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, db.Close())

		value, err := db.Get("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrClosed)
	})

	t.Run("returns ErrNotFound if key does not exist", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		value, err := db.Get("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestEmbeddedDBDelete(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, db.Close())

		assert.ErrorIs(t, db.Delete("key"), ErrClosed)
	})
}
