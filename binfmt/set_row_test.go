package binfmt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from encoding key", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.SetRow[Binary](string([]byte{0xff, 0xfe, 0xfd}), []byte("value"))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "encode key")
	})

	t.Run("encodes set row key and uint value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := uint64(30)
		enc := NewEncoder()

		err := enc.SetRow[Uint](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtUint Uint

		assert.Equal(t, []byte{
			vtUint.Ident(),
			0b0001_1110,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes set row key and int value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := int64(30)
		enc := NewEncoder()

		err := enc.SetRow[Int](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtInt Int

		assert.Equal(t, []byte{
			vtInt.Ident(),
			0b0011_1100,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes set row key and binary value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}
		enc := NewEncoder()

		err := enc.SetRow[Binary](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 4)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtBinary Binary

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			vtBinary.Ident(),
			0b0000_0110,
		}, enc.writes[2].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err = io.ReadAll(enc.writes[3].(*bytes.Reader))
		assert.NoError(t, err)
		assert.Equal(t, value, buf)
	})

	t.Run("encodes set row key and string value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := enc.SetRow[String](key, key)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 4)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtString String

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			vtString.Ident(),
			0b0001_1110,
		}, enc.writes[2].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err = io.ReadAll(enc.writes[3].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, key, string(buf))
	})
}

func TestDecoderSetRow(t *testing.T) {
	t.Parallel()

	var vtString String

	t.Run("returns error from String", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid string length.
		var buf bytes.Buffer
		for range 10 {
			buf.WriteByte(0xff)
		}

		dec := NewDecoder(&buf)
		key, value, n, err := dec.SetRow[Binary]()

		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode key")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF while decoding value identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by omitting the value identifier and value.
		enc := NewEncoder()
		err := vtString.Encode(enc, "key")
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		key, value, n, err := dec.SetRow[Binary]()

		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 4, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "decode value type identifier")
	})

	t.Run("return ErrRowValueIdentInvalid if decoded an invalid value identifier", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtString.Encode(enc, "key")
		assert.NoError(t, err)

		// Force error by writing invalid value identifier.
		enc.Buffer.WriteByte('Z')

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		key, value, n, err := dec.SetRow[Binary]()

		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, ErrRowValueIdentInvalid)
	})

	t.Run("return error from Binary for binary value identifier", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtString.Encode(enc, "key")
		assert.NoError(t, err)

		var vtBinary Binary
		enc.Buffer.WriteByte(vtBinary.Ident())

		// Force error by writing invalid binary length.
		for range 10 {
			enc.Buffer.WriteByte(0xff)
		}

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		key, value, n, err := dec.SetRow[Binary]()

		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 4+1+10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode value")
	})

	t.Run("returns the key and uint value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := uint64(30)
		enc := NewEncoder()

		err := vtString.Encode(enc, key)
		assert.NoError(t, err)

		vtUint := Uint{}
		enc.Buffer.WriteByte(vtUint.Ident())
		err = vtUint.Encode(enc, value)

		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actualKey, actualValue, n, err := dec.SetRow[Uint]()
		assert.NoError(t, err)

		assert.Equal(t, 16+1+1, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns the key and int value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := int64(30)
		enc := NewEncoder()

		err := vtString.Encode(enc, key)
		assert.NoError(t, err)

		vtInt := Int{}
		enc.Buffer.WriteByte(vtInt.Ident())
		err = vtInt.Encode(enc, value)

		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actualKey, actualValue, n, err := dec.SetRow[Int]()
		assert.NoError(t, err)

		assert.Equal(t, 16+1+1, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns the key and binary value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}
		enc := NewEncoder()

		err := vtString.Encode(enc, key)
		assert.NoError(t, err)

		var vtBinary Binary
		enc.Buffer.WriteByte(vtBinary.Ident())
		err = vtBinary.Encode(enc, value)

		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actualKey, actualValue, n, err := dec.SetRow[Binary]()
		assert.NoError(t, err)

		assert.Equal(t, 16+1+4, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, value, actualValue)
	})

	t.Run("returns the key and string value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := vtString.Encode(enc, key)
		assert.NoError(t, err)

		var vtString String
		enc.Buffer.WriteByte(vtString.Ident())
		err = vtString.Encode(enc, key)

		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actualKey, actualValue, n, err := dec.SetRow[String]()
		assert.NoError(t, err)

		assert.Equal(t, 16+1+16, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, key, actualValue)
	})
}
