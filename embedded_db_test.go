package kvega

import (
	"bytes"
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

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		assert.Equal(t, path, db.path)
		assert.NotNil(t, db.file)
		assert.Equal(t, path, db.file.Name())
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

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		assert.ErrorIs(t, db.Set("", []byte("value")), ErrEmptyKey)
	})

	t.Run("writes a new row to the end of the file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		_, err = db.file.Write([]byte("D,a2V5\n"))
		assert.NoError(t, err)

		_, err = db.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		err = db.Set("key", []byte("value"))
		assert.NoError(t, err)

		_, err = db.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		var buf bytes.Buffer

		_, err = io.Copy(&buf, db.file)
		assert.NoError(t, err)

		assert.Equal(t, "D,a2V5\nS,a2V5,dmFsdWU=\n", buf.String())
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

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		assert.ErrorIs(t, db.Delete(""), ErrEmptyKey)
	})

	t.Run("writes a new row to the end of the file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		_, err = db.file.Write([]byte("S,a2V5,dmFsdWU=\n"))
		assert.NoError(t, err)

		_, err = db.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		err = db.Delete("key")
		assert.NoError(t, err)

		_, err = db.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		var buf bytes.Buffer

		_, err = io.Copy(&buf, db.file)
		assert.NoError(t, err)

		assert.Equal(t, "S,a2V5,dmFsdWU=\nD,a2V5\n", buf.String())
	})
}
