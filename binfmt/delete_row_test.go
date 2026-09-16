package binfmt

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from WriteString", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteDeleteRow(&buf, string([]byte{0xff, 0xfe, 0xfd}))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "write key")
	})

	t.Run("writes delete row to writer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		key := "こんにちは"

		err := WriteDeleteRow(&buf, key)
		assert.NoError(t, err)

		assert.Equal(t, 1+1+15, buf.Len())
		assert.Equal(t, byte(DeleteRowIdent), buf.Bytes()[0])
		assert.Equal(t, byte(0b0001_1110), buf.Bytes()[1])
		assert.Equal(t, []byte(key), buf.Bytes()[2:17])
	})
}

func TestReadDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from ReadString", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid string length.
		var buf bytes.Buffer
		for range 10 {
			buf.WriteByte(0xff)
		}

		key, n, err := ReadDeleteRow(bufio.NewReader(&buf))
		assert.Empty(t, key)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrReadIntInvalid)
		assert.ErrorContains(t, err, "read key")
	})

	t.Run("returns the key", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"

		var buf bytes.Buffer

		err := WriteString(&buf, key)
		assert.NoError(t, err)

		actualKey, n, err := ReadDeleteRow(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 16, n)
		assert.Equal(t, key, actualKey)
	})
}
