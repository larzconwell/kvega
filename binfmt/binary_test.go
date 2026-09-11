package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteBinary(t *testing.T) {
	t.Parallel()

	t.Run("returns error from WriteUint", func(t *testing.T) {
		t.Parallel()

		// Error after the one byte was written for the length.
		n, err := WriteBinary(&errReadWriter{err: io.ErrClosedPipe, errAfter: 1}, []byte("test"))
		assert.Equal(t, 1, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write uint")
	})

	t.Run("returns error from copying value", func(t *testing.T) {
		t.Parallel()

		// Error after the length was written and the beginning of the value is being written.
		n, err := WriteBinary(&errReadWriter{err: io.ErrClosedPipe, errAfter: 3}, []byte("test"))
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write binary")
	})

	t.Run("writes length and value to writer", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		var buf bytes.Buffer

		n, err := WriteBinary(&buf, value)
		assert.NoError(t, err)

		assert.Equal(t, len(value)+2, n)
		assert.Equal(t, []byte{
			0b1000_0001,
			0,
		}, buf.Bytes()[:2])
		assert.Equal(t, value, buf.Bytes()[2:])
	})
}

func TestReadBinary(t *testing.T) {
	t.Parallel()

	t.Run("returns error from ReadUint", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadBinary(bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
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

		value, n, err := ReadBinary(bufio.NewReader(&buf))
		assert.Empty(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "read binary")
	})

	t.Run("returns error from reading value", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint(5))
		assert.NoError(t, err)

		buf.Write(make([]byte, 5))

		value, n, err := ReadBinary(bufio.NewReader(&errReadWriter{
			buf:      &buf,
			err:      io.ErrClosedPipe,
			errAfter: 5,
		}))

		assert.Empty(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read binary")
	})

	t.Run("returns the read value", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		var buf bytes.Buffer

		_, err := WriteUint(&buf, uint64(len(value)))
		assert.NoError(t, err)

		buf.Write(value)

		actual, n, err := ReadBinary(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, len(value)+2, n)
		assert.Equal(t, value, actual)
	})
}
