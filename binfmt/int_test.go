package binfmt

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntIdent(t *testing.T) {
	t.Parallel()

	var vtInt Int
	assert.Equal(t, byte('I'), vtInt.Ident())
}

func TestIntEncode(t *testing.T) {
	t.Parallel()

	var vtInt Int

	t.Run("encodes zero in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 0)
		assert.NoError(t, err)

		assert.Equal(t, []byte{0}, enc.Buffer.Bytes())
	})

	t.Run("encodes small positive int in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 50)
		assert.NoError(t, err)

		assert.Equal(t, []byte{0b0110_0100}, enc.Buffer.Bytes())
	})

	t.Run("encodes small negative int in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, -50)
		assert.NoError(t, err)

		assert.Equal(t, []byte{0b0110_0011}, enc.Buffer.Bytes())
	})

	t.Run("encodes medium positive int in three bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 48_059)
		assert.NoError(t, err)

		assert.Equal(t, []byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0110,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes medium negative int in three bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, -48_059)
		assert.NoError(t, err)

		assert.Equal(t, []byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0101,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes large positive int in nine bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 845_475_770_045_021_115)
		assert.NoError(t, err)

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
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes large negative int in nine bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, -845_475_770_045_021_115)
		assert.NoError(t, err)

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
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes min int in ten bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, math.MinInt64)
		assert.NoError(t, err)

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
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes max int in ten bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, math.MaxInt64)
		assert.NoError(t, err)

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
		}, enc.Buffer.Bytes())
	})
}

func TestIntDecode(t *testing.T) {
	t.Parallel()

	var vtInt Int

	t.Run("return io.ErrUnexpectedEOF if encountering io.EOF before the last byte is read", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			// Expecting at least one more byte without continuation set.
		})

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("return error if encountered error reading bytes", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(&errReadWriter{err: io.ErrClosedPipe})
		value, n, err := vtInt.Decode(dec)

		assert.Zero(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("return error if reached ten byte cap without encountering unset continuation bit", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
	})

	t.Run("return error if decoded int is larger than fits in int64", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0010,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0b1000_0000,
			0,
		})

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeIntOverflow)
	})

	t.Run("decodes zero zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0)

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, int64(0), value)
	})

	t.Run("decodes small positive zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0110_0100)

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, int64(50), value)
	})

	t.Run("decodes small negative zigzag encoded int in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0110_0011)

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, int64(-50), value)
	})

	t.Run("decodes medium positive zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0110,
		})

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, int64(48_059), value)
	})

	t.Run("decodes medium negative zigzag encoded int in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0101,
			0b1110_1110,
			0b0111_0101,
		})

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, int64(-48_059), value)
	})

	t.Run("decodes large positive zigzag encoded int in nine bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, int64(845_475_770_045_021_115), value)
	})

	t.Run("decodes large negative zigzag encoded int in nine bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, int64(-845_475_770_045_021_115), value)
	})

	t.Run("decodes min zigzag encoded int in ten bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, int64(math.MinInt64), value)
	})

	t.Run("decodes max zigzag encoded int in ten bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtInt.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, int64(math.MaxInt64), value)
	})
}
