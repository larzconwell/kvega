package binfmt

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUintIdent(t *testing.T) {
	t.Parallel()

	var vtUint = Uint{}
	assert.Equal(t, byte('U'), vtUint.Ident())
}

func TestUintEncode(t *testing.T) {
	t.Parallel()

	var vtUint = Uint{}

	t.Run("encodes zero in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtUint.Encode(enc, 0)
		assert.NoError(t, err)

		assert.Equal(t, []byte{0}, enc.Buffer.Bytes())
	})

	t.Run("encodes small uint in one byte", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtUint.Encode(enc, 127)
		assert.NoError(t, err)

		assert.Equal(t, []byte{0b0111_1111}, enc.Buffer.Bytes())
	})

	t.Run("encodes medium uint in three bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtUint.Encode(enc, 0xbbbb)
		assert.NoError(t, err)

		assert.Equal(t, []byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes large uint in nine bytes", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtUint.Encode(enc, 0xbbb_bbbb_bbbb_bbbb)
		assert.NoError(t, err)

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
		err := vtUint.Encode(enc, math.MaxUint64)
		assert.NoError(t, err)

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

func TestUintDecode(t *testing.T) {
	t.Parallel()

	var vtUint = Uint{}

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
		value, n, err := vtUint.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("return error if encountered error reading bytes", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(&errReadWriter{err: io.ErrClosedPipe})
		value, n, err := vtUint.Decode(dec)

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
		value, n, err := vtUint.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeUintInvalid)
	})

	t.Run("return error if decoded uint is larger than fits in uint64", func(t *testing.T) {
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
		value, n, err := vtUint.Decode(dec)

		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeUintOverflow)
	})

	t.Run("decodes zero encoded in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0)

		dec := NewDecoder(&buf)
		value, n, err := vtUint.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, uint64(0), value)
	})

	t.Run("decodes small uint encoded in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0111_1111)

		dec := NewDecoder(&buf)
		value, n, err := vtUint.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, uint64(127), value)
	})

	t.Run("decodes medium uint encoded in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		})

		dec := NewDecoder(&buf)
		value, n, err := vtUint.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, uint64(0xbbbb), value)
	})

	t.Run("decodes large uint encoded in nine bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtUint.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, uint64(0xbbb_bbbb_bbbb_bbbb), value)
	})

	t.Run("decodes max uint encoded in 10 bytes", func(t *testing.T) {
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

		dec := NewDecoder(&buf)
		value, n, err := vtUint.Decode(dec)
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, uint64(math.MaxUint64), value)
	})
}
