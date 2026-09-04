package kvega

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzEmbeddedDB(f *testing.F) {
	f.Add("key", []byte("value"))

	f.Fuzz(func(t *testing.T, key string, value []byte) {
		if key == "" {
			return
		}

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		err = db.Set(key, value)
		assert.NoError(t, err)

		retrievedValue, err := db.Get(key)
		assert.NoError(t, err)

		assert.Equal(t, value, retrievedValue)
	})
}

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

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		value, err := db.Get("")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrEmptyKey)
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

	t.Run("returns io.ErrUnexpectedEOF if reading a line without a newline", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		// Force error by omitting the final newline.
		_, err = db.file.Write([]byte("D,a2V5"))
		assert.NoError(t, err)

		value, err := db.Get("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("returns InvalidRowError(ErrInvalidRowType) if encountering a row with an invalid row type", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		db, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, db.Close())
		}()

		// Force error by using an invalid first column.
		_, err = db.file.Write([]byte("X,a2V5\n"))
		assert.NoError(t, err)

		value, err := db.Get("key")
		assert.Nil(t, value)

		var ire *InvalidRowError
		assert.ErrorAs(t, err, &ire)
		assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowType)
	})

	t.Run("reading set data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns InvalidRowError(ErrInvalidRowColumns) if encountering a row with invalid number of columns", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by omitting the , and subsequent value.
			_, err = db.file.Write([]byte("S,a2V5\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumns)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with an empty key column", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by omitting the key column.
			_, err = db.file.Write([]byte("S,,dmFsdWU=\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a key that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by using invalid base64 in key column.
			_, err = db.file.Write([]byte("S,****,dmFsdWU=\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a value that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by using invalid base64 in value column.
			_, err = db.file.Write([]byte("S,a2V5,****\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns the value stored in the row for the key", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			_, err = db.file.Write([]byte("S,a2V5,dmFsdWU=\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("value"), value)
		})
	})

	t.Run("reading delete data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns InvalidRowError(ErrInvalidRowColumns) if encountering a row with invalid number of columns", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = db.file.Write([]byte("D\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumns)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with an empty key column", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = db.file.Write([]byte("D,\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a key that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = db.file.Write([]byte("D,****\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns ErrNotFound when a delete row is found for the key", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			_, err = db.file.Write([]byte("D,a2V5\n"))
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	})

	t.Run("reading mixed data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the result of the last row found for the key assuming the last row was a set", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			contents := bytes.Join([][]byte{
				[]byte("S,a2V5,dmFsdWU=\n"),
				[]byte("S,a2V5Mg==,dmFsdWU=\n"),
				[]byte("D,a2V5\n"),
				[]byte("S,a2V5,dmFsdWUy\n"),
			}, nil)

			_, err = db.file.Write(contents)
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("value2"), value)
		})

		t.Run("returns the result of the last row found for the key assuming the last row was a delete", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			db, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, db.Close())
			}()

			contents := bytes.Join([][]byte{
				[]byte("S,a2V5,dmFsdWU=\n"),
				[]byte("S,a2V5Mg==,dmFsdWU=\n"),
				[]byte("D,a2V5Mg==\n"),
				[]byte("S,a2V5,dmFsdWUy\n"),
				[]byte("D,a2V5\n"),
			}, nil)

			_, err = db.file.Write(contents)
			assert.NoError(t, err)

			value, err := db.Get("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, ErrNotFound)
		})
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
