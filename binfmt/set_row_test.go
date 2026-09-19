package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from encoding key", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.SetRow(string([]byte{0xff, 0xfe, 0xfd}), []byte("value"))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "encode key")
	})

	t.Run("encodes set rows key and value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}
		enc := NewEncoder()

		err := enc.SetRow(key, value)
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

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			BinaryIdent,
			0b0000_0110,
		}, enc.writes[2].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err = io.ReadAll(enc.writes[3].(*bytes.Reader))
		assert.NoError(t, err)
		assert.Equal(t, value, buf)
	})
}

func TestReadSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from ReadString", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid string length.
		var buf bytes.Buffer
		for range 10 {
			buf.WriteByte(0xff)
		}

		key, value, n, err := ReadSetRow(bufio.NewReader(&buf))
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrReadIntInvalid)
		assert.ErrorContains(t, err, "read key")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF while reading value identifier", func(t *testing.T) {
		t.Parallel()

		// Force error by omitting the value identifier and value.
		enc := NewEncoder()
		err := enc.String("key")
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		key, value, n, err := ReadSetRow(reader)
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 4, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "read value identifier")
	})

	t.Run("return ErrRowValueIdentInvalid if read an invalid value identifier", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.String("key")
		assert.NoError(t, err)

		// Force error by writing invalid value identifier.
		enc.Buffer.WriteByte('Z')

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		key, value, n, err := ReadSetRow(reader)
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, ErrRowValueIdentInvalid)
	})

	t.Run("return error from ReadBinary for binary value identifier", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.String("key")
		assert.NoError(t, err)

		enc.Buffer.WriteByte(BinaryIdent)

		// Force error by writing invalid binary length.
		for range 10 {
			enc.Buffer.WriteByte(0xff)
		}

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		key, value, n, err := ReadSetRow(reader)
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 4+1+10, n)
		assert.ErrorIs(t, err, ErrReadIntInvalid)
		assert.ErrorContains(t, err, "read binary value")
	})

	t.Run("returns the key and binary value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}
		enc := NewEncoder()

		err := enc.String(key)
		assert.NoError(t, err)

		enc.Buffer.WriteByte(BinaryIdent)
		enc.Binary(value)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		actualKey, actualValue, n, err := ReadSetRow(reader)
		assert.NoError(t, err)

		assert.Equal(t, 16+1+4, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, value, actualValue)
	})
}
