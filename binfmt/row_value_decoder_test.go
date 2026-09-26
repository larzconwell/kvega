package binfmt

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRowValueDecoderGet(t *testing.T) {
	t.Parallel()

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF while decoding value type identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by omitting the value type identifier and value.
		enc := NewEncoder()

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		value, n, err := rvd.Get[Binary]()

		assert.Nil(t, value)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "decode value type identifier")
	})

	t.Run("return ErrRowValueIdentInvalid if decoded an invalid value type identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid value type identifier.
		enc := NewEncoder()
		enc.Buffer.WriteByte('Z')

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		value, n, err := rvd.Get[Binary]()

		assert.Nil(t, value)
		assert.Equal(t, 1, n)
		assert.ErrorIs(t, err, ErrRowValueIdentInvalid)
	})

	t.Run("return error from Binary for binary value type identifier", func(t *testing.T) {
		t.Parallel()

		var vtBinary Binary

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtBinary.Ident())

		// Force error by writing invalid binary length.
		for range 10 {
			enc.Buffer.WriteByte(0xff)
		}

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		value, n, err := rvd.Get[Binary]()

		assert.Nil(t, value)
		assert.Equal(t, 1+10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode value")
	})

	t.Run("returns rows uint value", func(t *testing.T) {
		t.Parallel()

		var vtUint Uint

		value := uint64(30)

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtUint.Ident())

		err := vtUint.Encode(enc, value)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		actualValue, n, err := rvd.Get[Uint]()
		assert.NoError(t, err)

		assert.Equal(t, 1+1, n)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns rows int value", func(t *testing.T) {
		t.Parallel()

		var vtInt Int

		value := int64(30)

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtInt.Ident())

		err := vtInt.Encode(enc, value)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		actualValue, n, err := rvd.Get[Int]()
		assert.NoError(t, err)

		assert.Equal(t, 1+1, n)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns rows binary value", func(t *testing.T) {
		t.Parallel()

		var vtBinary Binary

		value := []byte{0xff, 0xfe, 0xfd}

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtBinary.Ident())

		err := vtBinary.Encode(enc, value)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		actualValue, n, err := rvd.Get[Binary]()
		assert.NoError(t, err)

		assert.Equal(t, 1+1+3, n)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns rows string value", func(t *testing.T) {
		t.Parallel()

		var vtString String

		value := "こんにちは"

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtString.Ident())

		err := vtString.Encode(enc, value)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		actualValue, n, err := rvd.Get[String]()
		assert.NoError(t, err)

		assert.Equal(t, 1+1+15, n)
		assert.Equal(t, value, actualValue)
	})
}
