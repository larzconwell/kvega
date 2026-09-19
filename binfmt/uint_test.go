package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderUint(t *testing.T) {
	t.Parallel()

	t.Run("encodes zero in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Uint(0)

		assert.Equal(t, []byte{0}, enc.Buffer.Bytes())
	})

	t.Run("encodes small uint in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Uint(127)

		assert.Equal(t, []byte{0b0111_1111}, enc.Buffer.Bytes())
	})

	t.Run("encodes medium uint in three bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Uint(0xbbbb)

		assert.Equal(t, []byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes large uint in nine bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Uint(0xbbb_bbbb_bbbb_bbbb)

		assert.Equal(t, []byte{
			0b1000_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b0011_1011,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes max uint in ten bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Uint(math.MaxUint64)

		assert.Equal(t, []byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b0111_1111,
		}, enc.Buffer.Bytes())
	})
}

func TestReadUint(t *testing.T) {
	t.Parallel()

	t.Run("return io.ErrUnexpectedEOF if encountering io.EOF before the last byte is read", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			// Expecting at least one more byte without continuation set.
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("return error if encountered error reading bytes", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadUint[uint64](bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
		assert.Zero(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("return error if reached ten byte cap without encounting unset continuation bit", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff, // Should be 0b0111_1111 instead
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrReadUintInvalid)
	})

	t.Run("return error if read uint is larger than fits in given uint type", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint8](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 2, n)
		assert.ErrorIs(t, err, ErrReadUintOverflow)
	})

	t.Run("reads zero encoded in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0)

		value, n, err := ReadUint[uint](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, uint(0), value)
	})

	t.Run("reads small uint encoded in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0111_1111)

		value, n, err := ReadUint[uint](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, uint(127), value)
	})

	t.Run("reads medium uint encoded in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, uint(0xbbbb), value)
	})

	t.Run("reads large uint encoded in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, uint64(0xbbb_bbbb_bbbb_bbbb), value)
	})

	t.Run("reads marg uint encoded in 10 bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b0111_1111,
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, uint64(math.MaxUint64), value)
	})
}
