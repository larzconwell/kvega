package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteInt(t *testing.T) {
	t.Parallel()

	t.Run("writes zero zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, 0)

		assert.Equal(t, []byte{0}, buf.Bytes())
	})

	t.Run("writes small positive zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, 50)

		assert.Equal(t, []byte{0b0110_0100}, buf.Bytes())
	})

	t.Run("writes small negative zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, -50)

		assert.Equal(t, []byte{0b0110_0011}, buf.Bytes())
	})

	t.Run("writes medium positive zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, 48_059)

		assert.Equal(t, []byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0110,
		}, buf.Bytes())
	})

	t.Run("writes medium negative zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, -48_059)

		assert.Equal(t, []byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0101,
		}, buf.Bytes())
	})

	t.Run("writes large positive zigzag encoded int in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, int64(845_475_770_045_021_115))

		assert.Equal(t, []byte{
			0b1001_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b0111_0110,
		}, buf.Bytes())
	})

	t.Run("writes large negative zigzag encoded int in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, int64(-845_475_770_045_021_115))

		assert.Equal(t, []byte{
			0b10010111,
			0b10111011,
			0b11011101,
			0b11101110,
			0b11110111,
			0b10111011,
			0b11011101,
			0b11101110,
			0b01110101,
		}, buf.Bytes())
	})

	t.Run("writes minimum zigzag encoded int in ten bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, math.MinInt64)

		assert.Equal(t, []byte{
			0b10000001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b01111111,
		}, buf.Bytes())
	})

	t.Run("writes maximum zigzag encoded int in ten bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		WriteInt(&buf, math.MaxInt64)

		assert.Equal(t, []byte{
			0b10000001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b01111110,
		}, buf.Bytes())
	})
}

func TestReadInt(t *testing.T) {
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

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("return error if encountered error reading bytes", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadInt[int64](bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
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

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrReadIntInvalid)
	})

	t.Run("return error if read int is larger than fits in given int type", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0110,
		})

		value, n, err := ReadInt[int8](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, ErrReadIntOverflow)
	})

	t.Run("reads zero zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0)

		value, n, err := ReadInt[int](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, 0, value)
	})

	t.Run("reads small positive zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0110_0100)

		value, n, err := ReadInt[int](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, 50, value)
	})

	t.Run("reads small negative zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0110_0011)

		value, n, err := ReadInt[int](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, -50, value)
	})

	t.Run("reads medium positive zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0110,
		})

		value, n, err := ReadInt[int](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, 48_059, value)
	})

	t.Run("reads medium negative zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0101,
		})

		value, n, err := ReadInt[int](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, -48_059, value)
	})

	t.Run("reads large positive zigzag encoded int in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1001_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b0111_0110,
		})

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, int64(845_475_770_045_021_115), value)
	})

	t.Run("reads large negative zigzag encoded int in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b10010111,
			0b10111011,
			0b11011101,
			0b11101110,
			0b11110111,
			0b10111011,
			0b11011101,
			0b11101110,
			0b01110101,
		})

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, int64(-845_475_770_045_021_115), value)
	})

	t.Run("writes minimum zigzag encoded int in ten bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b10000001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b01111111,
		})

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, int64(math.MinInt64), value)
	})

	t.Run("writes maximum zigzag encoded int in ten bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b10000001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b01111110,
		})

		value, n, err := ReadInt[int64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, int64(math.MaxInt64), value)
	})
}
