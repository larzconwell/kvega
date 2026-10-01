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

	t.Run("return ValueTypeMismatchError if decoded an unexpected value type identifier and unreads the value type identifier", func(t *testing.T) {
		t.Parallel()

		var vtBinary Binary

		// Force error by writing unexpected value type identifier.
		enc := NewEncoder()
		enc.Buffer.WriteByte('I')

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		value, n, err := rvd.Get[Binary]()

		assert.Nil(t, value)
		assert.Equal(t, 1, n)

		var vtmerr ValueTypeMismatchError
		assert.ErrorAs(t, err, &vtmerr)
		assert.Equal(t, vtBinary.Ident(), vtmerr.Expected)
		assert.Equal(t, byte('I'), vtmerr.Found)

		ident, err := rvd.dec.reader.ReadByte()
		assert.NoError(t, err)

		assert.Equal(t, byte('I'), ident)
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

	t.Run("returns rows binary value as a copy", func(t *testing.T) {
		t.Parallel()

		var (
			vtInt    Int
			vtBinary Binary
		)

		value := []byte{0xff, 0xfe, 0xfd}

		enc := NewEncoder()
		enc.Buffer.WriteByte(vtBinary.Ident())

		err := vtBinary.Encode(enc, value)
		assert.NoError(t, err)

		enc.Buffer.WriteByte(vtInt.Ident())

		err = vtInt.Encode(enc, 50)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		actualValue, n, err := rvd.Get[Binary]()
		assert.NoError(t, err)

		_, _, err = rvd.Get[Int]()
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

func TestRowValueDecoderSkip(t *testing.T) {
	t.Parallel()

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF while decoding value type identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by omitting the value type identifier and value.
		enc := NewEncoder()

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		n, err := rvd.Skip()

		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "decode value type identifier")
	})

	t.Run("return ErrRowValueIdentInvalid if decoded an invalid value type identifier and unreads the value type identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid value type identifier.
		enc := NewEncoder()
		enc.Buffer.WriteByte('Z')

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		rvd := &RowValueDecoder{dec: NewDecoder(reader)}
		n, err := rvd.Skip()

		assert.Equal(t, 1, n)
		assert.ErrorIs(t, err, ErrRowValueIdentInvalid)

		ident, err := rvd.dec.reader.ReadByte()
		assert.NoError(t, err)

		assert.Equal(t, byte('Z'), ident)
	})

	t.Run("returns error from decoding value types", func(t *testing.T) {
		t.Parallel()

		var (
			vtUint   Uint
			vtInt    Int
			vtBinary Binary
			vtString String
		)

		uintEnc := NewEncoder()
		intEnc := NewEncoder()
		binaryEnc := NewEncoder()
		stringEnc := NewEncoder()

		uintEnc.Buffer.WriteByte(vtUint.Ident())
		intEnc.Buffer.WriteByte(vtInt.Ident())
		binaryEnc.Buffer.WriteByte(vtBinary.Ident())
		stringEnc.Buffer.WriteByte(vtString.Ident())

		for range 10 {
			uintEnc.Buffer.WriteByte(0xff)
			intEnc.Buffer.WriteByte(0xff)
			binaryEnc.Buffer.WriteByte(0xff)
			stringEnc.Buffer.WriteByte(0xff)
		}

		cases := []struct {
			enc                 *Encoder
			expectedN           int
			expectedErr         error
			expectedErrContains string
		}{
			{
				enc:         uintEnc,
				expectedN:   1 + 10,
				expectedErr: ErrDecodeUintInvalid,
			},
			{
				enc:         intEnc,
				expectedN:   1 + 10,
				expectedErr: ErrDecodeIntInvalid,
			},
			{
				enc:                 binaryEnc,
				expectedN:           1 + 10,
				expectedErr:         ErrDecodeIntInvalid,
				expectedErrContains: "decode binary length",
			},
			{
				enc:                 stringEnc,
				expectedN:           1 + 10,
				expectedErr:         ErrDecodeIntInvalid,
				expectedErrContains: "decode string length",
			},
		}

		for _, test := range cases {
			reader, err := encoderToBufReader(test.enc)
			assert.NoError(t, err)

			rvd := &RowValueDecoder{dec: NewDecoder(reader)}
			n, err := rvd.Skip()

			assert.Equal(t, test.expectedN, n)
			assert.ErrorIs(t, err, test.expectedErr)
			assert.ErrorContains(t, err, "decode value")

			if test.expectedErrContains != "" {
				assert.ErrorContains(t, err, test.expectedErrContains)
			}
		}
	})

	t.Run("consumes value from decoders reader", func(t *testing.T) {
		t.Parallel()

		var (
			vtUint   Uint
			vtInt    Int
			vtBinary Binary
			vtString String
		)

		uintEnc := NewEncoder()
		intEnc := NewEncoder()
		binaryEnc := NewEncoder()
		stringEnc := NewEncoder()

		uintEnc.Buffer.WriteByte(vtUint.Ident())
		intEnc.Buffer.WriteByte(vtInt.Ident())
		binaryEnc.Buffer.WriteByte(vtBinary.Ident())
		stringEnc.Buffer.WriteByte(vtString.Ident())

		uintEnc.Buffer.WriteByte(1)
		intEnc.Buffer.WriteByte(2)
		binaryEnc.Buffer.WriteByte(10)
		stringEnc.Buffer.WriteByte(10)

		binaryEnc.Buffer.WriteString("value")
		stringEnc.Buffer.WriteString("value")

		cases := []struct {
			enc       *Encoder
			expectedN int
		}{
			{enc: uintEnc, expectedN: 1 + 1},
			{enc: intEnc, expectedN: 1 + 1},
			{enc: binaryEnc, expectedN: 1 + 1 + 5},
			{enc: stringEnc, expectedN: 1 + 1 + 5},
		}

		for _, test := range cases {
			test.enc.Buffer.WriteString("leftover")

			reader, err := encoderToBufReader(test.enc)
			assert.NoError(t, err)

			rvd := &RowValueDecoder{dec: NewDecoder(reader)}
			n, err := rvd.Skip()
			assert.NoError(t, err)

			assert.Equal(t, test.expectedN, n)

			data, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, "leftover", string(data))
		}
	})
}
