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

	t.Run("writes empty binary to writer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteBinary(&buf, nil)

		assert.Equal(t, []byte{0}, buf.Bytes())
	})

	t.Run("writes length and value to writer", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		var buf bytes.Buffer
		WriteBinary(&buf, value)

		assert.Equal(t, len(value)+2, buf.Len())
		assert.Equal(t, []byte{
			0b1000_0010,
			0,
		}, buf.Bytes()[:2])
		assert.Equal(t, value, buf.Bytes()[2:])
	})
}

func TestReadBinary(t *testing.T) {
	t.Parallel()

	t.Run("returns error from ReadInt", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadBinary(bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
		assert.Empty(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read int")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF before the value has been completely read", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, 5)
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
		WriteInt(&buf, 5)
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

	t.Run("returns empty binary from reader", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, 0)

		actual, n, err := ReadBinary(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, make([]byte, 0), actual)
	})

	t.Run("returns the read value", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		var buf bytes.Buffer
		WriteInt(&buf, len(value))
		buf.Write(value)

		actual, n, err := ReadBinary(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, len(value)+2, n)
		assert.Equal(t, value, actual)
	})
}
