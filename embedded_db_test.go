package kvega

import (
	"bytes"
	crypto "crypto/rand"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type command struct {
	typ string
	key string
}

func BenchmarkEmbeddedDBSet(b *testing.B) {
	path := filepath.Join(b.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	require.NoError(b, err)

	defer func() {
		require.NoError(b, edb.Close())
	}()

	// Using math/rand/v2 is good enough for this.
	//gosec:disable G404
	rand := rand.New(rand.NewPCG(0, 0))
	count := -1

	for b.Loop() {
		b.StopTimer()

		count++

		key := strconv.Itoa(count)
		value := make([]byte, rand.IntN(10_000))

		_, err := crypto.Read(value)
		require.NoError(b, err)
		b.StartTimer()

		require.NoError(b, edb.Set(key, value))
	}
}

func BenchmarkEmbeddedDBGet(b *testing.B) {
	path := filepath.Join(b.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	require.NoError(b, err)

	defer func() {
		require.NoError(b, edb.Close())
	}()

	// Using math/rand/v2 is good enough for this.
	//gosec:disable G404
	rand := rand.New(rand.NewPCG(0, 0))
	keys := 10_000

	for i := range keys {
		key := strconv.Itoa(i)
		value := make([]byte, rand.IntN(10_000))

		_, err := crypto.Read(value)
		require.NoError(b, err)
		b.StartTimer()

		require.NoError(b, edb.Set(key, value))
	}

	require.NoError(b, edb.buildIndex())

	for b.Loop() {
		b.StopTimer()

		key := strconv.Itoa(rand.IntN(keys))

		b.StartTimer()

		_, err := edb.Get(key)
		require.NoError(b, err)
	}
}

func BenchmarkParallelEmbeddedDBGet(b *testing.B) {
	path := filepath.Join(b.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	require.NoError(b, err)

	defer func() {
		require.NoError(b, edb.Close())
	}()

	// Using math/rand/v2 is good enough for this.
	//gosec:disable G404
	rand := rand.New(rand.NewPCG(0, 0))
	keys := 10_000

	for i := range keys {
		key := strconv.Itoa(i)
		value := make([]byte, rand.IntN(10_000))

		_, err := crypto.Read(value)
		require.NoError(b, err)
		b.StartTimer()

		require.NoError(b, edb.Set(key, value))
	}

	require.NoError(b, edb.buildIndex())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := strconv.Itoa(rand.IntN(keys))

			_, err := edb.Get(key)
			require.NoError(b, err)
		}
	})
}

func BenchmarkEmbeddedDBGetErrNotFound(b *testing.B) {
	path := filepath.Join(b.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	require.NoError(b, err)

	defer func() {
		require.NoError(b, edb.Close())
	}()

	// Using math/rand/v2 is good enough for this.
	//gosec:disable G404
	rand := rand.New(rand.NewPCG(0, 0))
	keys := 5_000

	for i := range keys {
		key := strconv.Itoa(i)
		value := make([]byte, rand.IntN(5_000))

		_, err := crypto.Read(value)
		require.NoError(b, err)
		b.StartTimer()

		require.NoError(b, edb.Set(key, value))
	}

	require.NoError(b, edb.buildIndex())

	for b.Loop() {
		_, err := edb.Get("not_found")
		require.ErrorIs(b, err, ErrNotFound)
	}
}

func BenchmarkEmbeddedDBBuildIndex(b *testing.B) {
	path := filepath.Join(b.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	require.NoError(b, err)

	defer func() {
		require.NoError(b, edb.Close())
	}()

	// Using math/rand/v2 is good enough for this.
	//gosec:disable G404
	rand := rand.New(rand.NewPCG(0, 0))

	for range 100_000 {
		key := strconv.Itoa(rand.IntN(70_000))
		require.NoError(b, edb.Set(key, []byte("value")))
	}

	for b.Loop() {
		require.NoError(b, edb.buildIndex())
	}
}

func FuzzEmbeddedDB(f *testing.F) {
	f.Add("key", []byte("value"))

	f.Fuzz(func(t *testing.T, key string, value []byte) {
		if key == "" {
			return
		}

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set(key, value))

		retrievedValue, err := edb.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, value, retrievedValue)
	})
}

func TestParallelEmbeddedDB(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.ArtifactDir(), "db.kvega")

	edb, err := OpenEmbeddedDB(path)
	assert.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, edb.Close())
	})

	workers := 4
	jobs := make(chan command, workers)
	sets := make(chan string, workers)
	kv := map[string][]byte{
		"one":   []byte("first"),
		"two":   []byte("second"),
		"three": []byte("third"),
		"four":  []byte("fourth"),
	}
	commands := []command{
		{typ: "set", key: "one"},
		{typ: "set", key: "two"},
		{typ: "set", key: "three"},
		{typ: "get"},
		{typ: "get"},
		{typ: "set", key: "four"},
		{typ: "get"},
		{typ: "get"},
	}

	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			for command := range jobs {
				switch command.typ {
				case "set":
					assert.NoError(t, edb.Set(command.key, kv[command.key]))

					sets <- command.key
				case "get":
					key := <-sets

					value, err := edb.Get(key)
					assert.NoError(t, err)
					assert.Equal(t, kv[key], value)
				}
			}
		})
	}

	for _, command := range commands {
		jobs <- command
	}

	close(jobs)

	wg.Wait()
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

		edb, err := OpenEmbeddedDB(path)
		assert.Nil(t, edb)
		assert.Error(t, err)
	})

	t.Run("returns error if failed to create file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		// Force error by creating the path as a directory.
		err := os.MkdirAll(path, 0o750)
		assert.NoError(t, err)

		edb, err := OpenEmbeddedDB(path)
		assert.Nil(t, edb)
		assert.Error(t, err)
	})

	t.Run("builds an index of keys to file offsets", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		assert.NoError(t, edb.Set("key", []byte("value")))
		assert.NoError(t, edb.Set("key2", []byte("value")))
		assert.NoError(t, edb.Delete("key"))
		assert.NoError(t, edb.Set("key", []byte("value2")))

		assert.NoError(t, edb.Close())

		// Reopen db to get the index built on open.
		edb, err = OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.Equal(t, map[string]int64{
			"key":  43,
			"key2": 16,
		}, edb.index)
	})

	t.Run("returns embedded db", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.Equal(t, path, edb.path)
		assert.Equal(t, path, edb.file.Name())
		assert.NotNil(t, edb.index)
		assert.NotNil(t, edb.file)
	})
}

func TestEmbeddedDBClose(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.ErrorIs(t, edb.Close(), ErrClosed)
	})

	t.Run("returns error if failed to sync file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		// Force error by manually closing file.
		assert.NoError(t, edb.file.Close())

		assert.Error(t, edb.Close())
	})

	t.Run("sets closed state", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.True(t, edb.closed.Load())
	})
}

func TestEmbeddedDBSet(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.ErrorIs(t, edb.Set("key", []byte("value")), ErrClosed)
	})

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.ErrorIs(t, edb.Set("", []byte("value")), ErrEmptyKey)
	})

	t.Run("writes a new row to the end of the file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Delete("key"))
		assert.NoError(t, edb.Set("key", []byte("value")))

		_, err = edb.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		var buf bytes.Buffer

		_, err = io.Copy(&buf, edb.file)
		assert.NoError(t, err)

		assert.Equal(t, "D,a2V5\nS,a2V5,dmFsdWU=\n", buf.String())
	})
}

func TestEmbeddedDBGet(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrClosed if database is closed", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		value, err := edb.Get("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrClosed)
	})

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		value, err := edb.Get("")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrEmptyKey)
	})

	t.Run("returns ErrNotFound if key does not exist", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		value, err := edb.Get("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns InvalidRowError(ErrInvalidRowType) if encountering a row with an invalid row type", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		// Force error by using an invalid first column.
		_, err = edb.file.Write([]byte("X,a2V5\n"))
		assert.NoError(t, err)

		value, err := edb.Get("key")
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

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by omitting the , and subsequent value.
			_, err = edb.file.Write([]byte("S,a2V5\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumns)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with an empty key column", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by omitting the key column.
			_, err = edb.file.Write([]byte("S,,dmFsdWU=\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a key that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by using invalid base64 in key column.
			_, err = edb.file.Write([]byte("S,****,dmFsdWU=\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a value that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by using invalid base64 in value column.
			_, err = edb.file.Write([]byte("S,a2V5,****\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns the value stored in the row for the key", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set("key", []byte("value")))

			value, err := edb.Get("key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("value"), value)
		})

		t.Run("returns the value stored in the row for the key regardless of the size of the value", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			var b byte

			value := make([]byte, 300*1024)
			for idx := range value {
				value[idx] = b
				b++
			}

			assert.NoError(t, edb.Set("key", value))

			actualValue, err := edb.Get("key")
			assert.NoError(t, err)
			assert.Equal(t, value, actualValue)
		})
	})

	t.Run("reading delete data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns InvalidRowError(ErrInvalidRowColumns) if encountering a row with invalid number of columns", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = edb.file.Write([]byte("D\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumns)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with an empty key column", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = edb.file.Write([]byte("D,\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns InvalidRowError(ErrInvalidRowColumn) if encountering a row with a key that's not encoded as expected", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by omitting the , and subsequent key.
			_, err = edb.file.Write([]byte("D,****\n"))
			assert.NoError(t, err)

			value, err := edb.Get("key")
			assert.Nil(t, value)

			var ire *InvalidRowError
			assert.ErrorAs(t, err, &ire)
			assert.ErrorIs(t, ire.Unwrap(), ErrInvalidRowColumn)
		})

		t.Run("returns ErrNotFound when a delete row is found for the key", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Delete("key"))

			value, err := edb.Get("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	})

	t.Run("reading mixed data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the result of the last row found for the key assuming the last row was a set", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set("key", []byte("value")))
			assert.NoError(t, edb.Set("key2", []byte("value")))
			assert.NoError(t, edb.Delete("key"))
			assert.NoError(t, edb.Set("key", []byte("value2")))

			value, err := edb.Get("key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("value2"), value)
		})

		t.Run("returns the result of the last row found for the key assuming the last row was a delete", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set("key", []byte("value")))
			assert.NoError(t, edb.Set("key2", []byte("value")))
			assert.NoError(t, edb.Delete("key2"))
			assert.NoError(t, edb.Set("key", []byte("value2")))
			assert.NoError(t, edb.Delete("key"))

			value, err := edb.Get("key")
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

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.ErrorIs(t, edb.Delete("key"), ErrClosed)
	})

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.ErrorIs(t, edb.Delete(""), ErrEmptyKey)
	})

	t.Run("writes a new row to the end of the file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set("key", []byte("value")))
		assert.NoError(t, edb.Delete("key"))

		_, err = edb.file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		var buf bytes.Buffer

		_, err = io.Copy(&buf, edb.file)
		assert.NoError(t, err)

		assert.Equal(t, "S,a2V5,dmFsdWU=\nD,a2V5\n", buf.String())
	})
}
