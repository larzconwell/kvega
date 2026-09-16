package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from WriteString", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteSetRow(&buf, string([]byte{0xff, 0xfe, 0xfd}), []byte("value"))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "write key")
	})

	t.Run("writes set row to writer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}

		err := WriteSetRow(&buf, key, value)
		assert.NoError(t, err)

		assert.Equal(t, 1+1+15+1+1+3, buf.Len())
		assert.Equal(t, byte(SetRowIdent), buf.Bytes()[0])
		assert.Equal(t, byte(0b0001_1110), buf.Bytes()[1])
		assert.Equal(t, []byte(key), buf.Bytes()[2:17])
		assert.Equal(t, byte(BinaryIdent), buf.Bytes()[17])
		assert.Equal(t, byte(0b0000_0110), buf.Bytes()[18])
		assert.Equal(t, value, buf.Bytes()[19:])
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

		var buf bytes.Buffer

		// Force error by omitting the value identifier and value.
		err := WriteString(&buf, "key")
		assert.NoError(t, err)

		key, value, n, err := ReadSetRow(bufio.NewReader(&buf))
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 4, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "read value identifier")
	})

	t.Run("return ErrRowValueIdentInvalid if read an invalid value identifier", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteString(&buf, "key")
		assert.NoError(t, err)

		// Force error by writing invalid value identifier.
		buf.WriteByte('Z')

		key, value, n, err := ReadSetRow(bufio.NewReader(&buf))
		assert.Empty(t, key)
		assert.Nil(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, ErrRowValueIdentInvalid)
	})

	t.Run("return error from ReadBinary for binary value identifier", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteString(&buf, "key")
		assert.NoError(t, err)

		buf.WriteByte(BinaryIdent)

		// Force error by writing invalid binary length.
		for range 10 {
			buf.WriteByte(0xff)
		}

		key, value, n, err := ReadSetRow(bufio.NewReader(&buf))
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

		var buf bytes.Buffer

		err := WriteString(&buf, key)
		assert.NoError(t, err)

		buf.WriteByte(BinaryIdent)

		WriteBinary(&buf, value)

		actualKey, actualValue, n, err := ReadSetRow(bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 16+1+4, n)
		assert.Equal(t, key, actualKey)
		assert.Equal(t, value, actualValue)
	})
}
