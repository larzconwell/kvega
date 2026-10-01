package kvega

import (
	crypto "crypto/rand"
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/larzconwell/kvega/binfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type row struct {
	ident byte
	key   string
	value []byte
}

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

	b.ResetTimer()

	for b.Loop() {
		b.StopTimer()

		count++

		key := strconv.Itoa(count)
		value := make([]byte, rand.IntN(10_000))

		_, err := crypto.Read(value)
		require.NoError(b, err)
		b.StartTimer()

		require.NoError(b, edb.Set[Binary](key, value))
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

		require.NoError(b, edb.Set[Binary](key, value))
	}

	b.ResetTimer()

	for b.Loop() {
		b.StopTimer()

		key := strconv.Itoa(rand.IntN(keys))

		b.StartTimer()

		_, err := edb.Get[Binary](key)
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

		require.NoError(b, edb.Set[Binary](key, value))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := strconv.Itoa(rand.IntN(keys))

			_, err := edb.Get[Binary](key)
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

		require.NoError(b, edb.Set[Binary](key, value))
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := edb.Get[Binary]("not_found")
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
		require.NoError(b, edb.Set[Binary](key, []byte("value")))
	}

	b.ResetTimer()

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

		if !utf8.ValidString(key) {
			return
		}

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set[Binary](key, value))

		retrievedValue, err := edb.Get[Binary](key)
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
					assert.NoError(t, edb.Set[Binary](command.key, kv[command.key]))

					sets <- command.key
				case "get":
					key := <-sets

					value, err := edb.Get[Binary](key)
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

	t.Run("returns error from buildIndex", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		// The caller should ensure the provided path is safe to open.
		//gosec:disable G304
		file, err := os.Create(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, file.Close())
		}()

		// Force error by writing invalid row identifier.
		_, err = file.WriteString("Z")
		assert.NoError(t, err)

		err = file.Sync()
		assert.NoError(t, err)

		edb, err := OpenEmbeddedDB(path)
		assert.Nil(t, edb)
		assert.ErrorIs(t, err, binfmt.ErrRowIdentInvalid)
		assert.ErrorContains(t, err, "decode row")
	})

	t.Run("builds an index of keys to file offsets", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))
		assert.NoError(t, edb.Set[Binary]("key2", []byte("value")))
		assert.NoError(t, edb.Delete("key"))
		assert.NoError(t, edb.Set[Binary]("key", []byte("value2")))

		assert.NoError(t, edb.Close())

		// Reopen db to get the index built on open.
		edb, err = OpenEmbeddedDB(path)
		assert.NoError(t, err)
		assert.NoError(t, edb.Close())

		assert.Equal(t, map[string]int64{
			"key":  30,
			"key2": 12,
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
		assert.NotNil(t, edb.index)
		assert.NotNil(t, edb.writer)
		assert.Equal(t, path, edb.writer.Name())
		assert.Len(t, edb.rmus, runtime.GOMAXPROCS(0))
		assert.Len(t, edb.readers, runtime.GOMAXPROCS(0))
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

	t.Run("returns error if failed to sync writer", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		// Force error by manually closing writer.
		assert.NoError(t, edb.writer.Close())
		// Manually clean up readers.
		for _, reader := range edb.readers {
			assert.NoError(t, reader.Close())
		}

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

		assert.ErrorIs(t, edb.Set[Binary]("key", []byte("value")), ErrClosed)
	})

	t.Run("returns ErrEmptyKey if key is empty", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.ErrorIs(t, edb.Set[Binary]("", []byte("value")), ErrEmptyKey)
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
		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))

		file := edb.readers[0]
		_, err = file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		actualRows, err := decodeRows(edb, file)
		assert.NoError(t, err)

		assert.Equal(t, []row{
			{
				ident: binfmt.DeleteRowIdent,
				key:   "key",
			},
			{
				ident: binfmt.SetRowIdent,
				key:   "key",
				value: []byte("value"),
			},
		}, actualRows)
	})

	t.Run("updates the index for the key", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Delete("key"))
		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))

		assert.Equal(t, int64(5), edb.index["key"])
	})

	t.Run("can successfully encode all supported value types", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set[Uint]("uint", 1))
		assert.NoError(t, edb.Set[Int]("int", 1))
		assert.NoError(t, edb.Set[Binary]("binary", []byte("value")))
		assert.NoError(t, edb.Set[String]("string", "你好"))
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

		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))
		assert.NoError(t, edb.Delete("key"))

		file := edb.readers[0]
		_, err = file.Seek(0, io.SeekStart)
		assert.NoError(t, err)

		actualRows, err := decodeRows(edb, file)
		assert.NoError(t, err)

		assert.Equal(t, []row{
			{
				ident: binfmt.SetRowIdent,
				key:   "key",
				value: []byte("value"),
			},
			{
				ident: binfmt.DeleteRowIdent,
				key:   "key",
			},
		}, actualRows)
	})

	t.Run("updates the index for the key", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))
		assert.NoError(t, edb.Delete("key"))

		assert.Equal(t, int64(12), edb.index["key"])
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

		value, err := edb.Get[Binary]("key")
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

		value, err := edb.Get[Binary]("")
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

		value, err := edb.Get[Binary]("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns error if encountering a row with invalid row identifier", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		// The caller should ensure the provided path is safe to open.
		//gosec:disable G304
		file, err := os.Create(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, file.Close())
		}()

		// Force error by writing invalid row identifier.
		_, err = file.WriteString("Z")
		assert.NoError(t, err)

		err = file.Sync()
		assert.NoError(t, err)

		value, err := edb.Get[Binary]("key")
		assert.Nil(t, value)
		assert.ErrorIs(t, err, binfmt.ErrRowIdentInvalid)
		assert.ErrorContains(t, err, "decode row")
	})

	t.Run("reads row at index offset if found for key", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))
		assert.NoError(t, edb.Set[Binary]("key", []byte("value2")))

		// Force a read for the first row to validate it uses the index.
		assert.NotEqual(t, 0, edb.index["key"])
		edb.index["key"] = 0

		value, err := edb.Get[Binary]("key")
		assert.NoError(t, err)
		assert.Equal(t, "value", string(value))
	})

	t.Run("reads all rows to find key if not found in the index", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.ArtifactDir(), "db.kvega")

		edb, err := OpenEmbeddedDB(path)
		assert.NoError(t, err)

		defer func() {
			assert.NoError(t, edb.Close())
		}()

		assert.NoError(t, edb.Set[Binary]("key", []byte("value")))
		assert.NoError(t, edb.Set[Binary]("key", []byte("value2")))

		// Empty out index to force unindexed path.
		assert.NotEmpty(t, edb.index)
		edb.index = nil

		value, err := edb.Get[Binary]("key")
		assert.NoError(t, err)
		assert.Equal(t, "value2", string(value))
	})

	t.Run("reading set data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns error if failed to read a set row completely", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			// Force error by only writing row identifier and key.
			enc := binfmt.NewEncoder()
			enc.Buffer.WriteByte(binfmt.SetRowIdent)

			var vtString binfmt.String

			err = vtString.Encode(enc, "key")
			assert.NoError(t, err)

			_, err = enc.WriteTo(edb.writer)
			assert.NoError(t, err)

			value, err := edb.Get[Binary]("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.ErrorContains(t, err, "decode value")
		})

		t.Run("returns error if value stored in row is different type than expected", func(t *testing.T) {
			t.Parallel()

			var (
				vtBinary Binary
				vtInt    Int
			)

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set[Binary]("key", []byte("value")))

			value, err := edb.Get[Int]("key")

			assert.Zero(t, value)
			assert.Equal(t, binfmt.ValueTypeMismatchError{
				Expected: vtInt.Ident(),
				Found:    vtBinary.Ident(),
			}, errors.Unwrap(err))
		})

		t.Run("returns the value stored in the row for the key", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set[Binary]("key", []byte("value")))

			value, err := edb.Get[Binary]("key")
			assert.NoError(t, err)
			assert.Equal(t, "value", string(value))
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

			assert.NoError(t, edb.Set[Binary]("key", value))

			actualValue, err := edb.Get[Binary]("key")
			assert.NoError(t, err)
			assert.Equal(t, value, actualValue)
		})
	})

	t.Run("reading delete data", func(t *testing.T) {
		t.Parallel()

		t.Run("returns error if failed to read a delete row completely", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			var vtInt binfmt.Int

			// Force error by only writing row identifier and the length portion of the key.
			enc := binfmt.NewEncoder()
			enc.Buffer.WriteByte(binfmt.DeleteRowIdent)

			err = vtInt.Encode(enc, 5)
			assert.NoError(t, err)

			_, err = enc.WriteTo(edb.writer)
			assert.NoError(t, err)

			value, err := edb.Get[Binary]("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.ErrorContains(t, err, "decode row")
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

			value, err := edb.Get[Binary]("key")
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

			assert.NoError(t, edb.Set[Binary]("key", []byte("abc")))
			assert.NoError(t, edb.Set[Binary]("key2", []byte("def")))
			assert.NoError(t, edb.Delete("key"))
			assert.NoError(t, edb.Set[Binary]("key", []byte("ghi")))
			assert.NoError(t, edb.Set[Binary]("key2", []byte("jkl")))

			// Empty out index to force unindexed path.
			assert.NotEmpty(t, edb.index)
			edb.index = nil

			value, err := edb.Get[Binary]("key")
			assert.NoError(t, err)
			assert.Equal(t, "ghi", string(value))
		})

		t.Run("returns the result of the last row found for the key assuming the last row was a delete", func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.ArtifactDir(), "db.kvega")

			edb, err := OpenEmbeddedDB(path)
			assert.NoError(t, err)

			defer func() {
				assert.NoError(t, edb.Close())
			}()

			assert.NoError(t, edb.Set[Binary]("key", []byte("abc")))
			assert.NoError(t, edb.Set[Binary]("key2", []byte("def")))
			assert.NoError(t, edb.Delete("key2"))
			assert.NoError(t, edb.Set[Binary]("key", []byte("ghi")))
			assert.NoError(t, edb.Delete("key"))
			assert.NoError(t, edb.Set[Binary]("key2", []byte("jkl")))

			// Empty out index to force unindexed path.
			assert.NotEmpty(t, edb.index)
			edb.index = nil

			value, err := edb.Get[Binary]("key")
			assert.Nil(t, value)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	})
}

func decodeRows(edb *EmbeddedDB, reader io.Reader) ([]row, error) {
	dec := binfmt.NewDecoder(reader)
	rows := make([]row, 0, 2)

	for {
		ident, key, valueDecoder, _, err := edb.decodeRow(dec)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		var value []byte

		if valueDecoder.HasValue {
			tmpValue, _, err := valueDecoder.Get[binfmt.Binary]()
			if err != nil {
				return nil, err
			}

			// Copy of value since it's only valid for this decodeRow loop.
			value = make([]byte, len(tmpValue))
			copy(value, tmpValue)
		}

		rows = append(rows, row{
			ident: ident,
			key:   key,
			value: value,
		})
	}

	return rows, nil
}
