package binfmt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from encoding key", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.DeleteRow(string([]byte{0xff, 0xfe, 0xfd}))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "encode key")
	})

	t.Run("encodes delete rows key", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := enc.DeleteRow(key)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			DeleteRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)
	})
}

func TestDecoderDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from String", func(t *testing.T) {
		t.Parallel()

		// Force error by writing invalid string length.
		var buf bytes.Buffer
		for range 10 {
			buf.WriteByte(0xff)
		}

		dec := NewDecoder(&buf)
		key, n, err := dec.DeleteRow()

		assert.Empty(t, key)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode key")
	})

	t.Run("returns the key", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := enc.String(key)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actualKey, n, err := dec.DeleteRow()
		assert.NoError(t, err)

		assert.Equal(t, 16, n)
		assert.Equal(t, key, actualKey)
	})
}
