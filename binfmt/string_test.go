package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteString(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrWriteStringInvalid when string is not valid UTF-8", func(t *testing.T) {
		t.Parallel()

		n, err := WriteString(nil, string([]byte{0xff, 0xfe, 0xfd}))
		assert.Zero(t, n)
		assert.ErrorIs(t, err, ErrWriteStringInvalid)
	})

	t.Run("returns error from WriteUint", func(t *testing.T) {
		t.Parallel()

		// Error after the one byte was written for the length.
		n, err := WriteString(&errReadWriter{err: io.ErrClosedPipe, errAfter: 1}, "test")
		assert.Equal(t, 1, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write uint")
	})

	t.Run("returns error from copying value", func(t *testing.T) {
		t.Parallel()

		// Error after the length was written and the beginning of the value is being written.
		n, err := WriteString(&errReadWriter{err: io.ErrClosedPipe, errAfter: 3}, "test")
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write string")
	})

	t.Run("writes length and value to writer", func(t *testing.T) {
		t.Parallel()

		value := "Cześć, こんにちは, 你好"

		var buf bytes.Buffer

		n, err := WriteString(&buf, value)
		assert.NoError(t, err)

		assert.Equal(t, len(value)+1, n)
		assert.Equal(t, byte(0b0010_0000), buf.Bytes()[0])
		assert.Equal(t, value, string(buf.Bytes()[1:]))
	})
}

func TestReadString(t *testing.T) {
	t.Parallel()

	t.Run("returns error from ReadUint", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadString(bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
		assert.Empty(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read uint")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF before the value has been completely read", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint(5))
		assert.NoError(t, err)

		buf.Write(make([]byte, 2))

		value, n, err := ReadString(bufio.NewReader(&buf))
		assert.Empty(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "read string")
	})

	t.Run("returns error from reading value", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint(5))
		assert.NoError(t, err)

		buf.Write(make([]byte, 5))

		value, n, err := ReadString(bufio.NewReader(&errReadWriter{
			buf:      &buf,
			err:      io.ErrClosedPipe,
			errAfter: 5,
		}))

		assert.Empty(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read string")
	})

	t.Run("returns ErrReadStringInvalid when read string is not valid UTF-8", func(t *testing.T) {
		t.Parallel()

		value := []byte{0xff, 0xfe, 0xfd}

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint64(len(value)))
		assert.NoError(t, err)

		buf.Write(value)

		actual, n, err := ReadString(bufio.NewReader(&buf))
		assert.Empty(t, actual)
		assert.Equal(t, len(value)+1, n)
		assert.ErrorIs(t, err, ErrReadStringInvalid)
	})

	t.Run("returns the read value", func(t *testing.T) {
		t.Parallel()

		value := "Cześć, こんにちは, 你好"

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint64(len(value)))
		assert.NoError(t, err)

		buf.WriteString(value)

		actual, n, err := ReadString(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, len(value)+1, n)
		assert.Equal(t, value, actual)
	})
}
