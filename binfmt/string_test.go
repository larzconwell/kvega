package binfmt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringIdent(t *testing.T) {
	t.Parallel()

	var vtString String
	assert.Equal(t, byte('S'), vtString.Ident())
}

func TestStringEncode(t *testing.T) {
	t.Parallel()

	var vtString String

	t.Run("returns ErrStringInvalid when string is not valid UTF-8", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtString.Encode(enc, string([]byte{0xff, 0xfe, 0xfd}))
		assert.ErrorIs(t, err, ErrStringInvalid)
	})

	t.Run("encodes empty string", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtString.Encode(enc, "")
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)
		assert.NotNil(t, enc.Buffer)
		assert.NotSame(t, enc.writes[0], enc.Buffer)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{0}, enc.writes[0].(*bytes.Buffer).Bytes())
		//nolint:forcetypeassert
		assert.Zero(t, enc.writes[1].(*strings.Reader).Size())
	})

	t.Run("encodes length and value", func(t *testing.T) {
		t.Parallel()

		value := "Cześć, こんにちは, 你好"
		enc := NewEncoder()

		err := vtString.Encode(enc, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)
		assert.NotNil(t, enc.Buffer)
		assert.NotSame(t, enc.writes[0], enc.Buffer)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			0b0100_0000,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, value, string(buf))
	})
}

func TestStringDecode(t *testing.T) {
	t.Parallel()

	var (
		vtInt    Int
		vtString String
	)

	t.Run("returns error from Int", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(&errReadWriter{err: io.ErrClosedPipe})
		value, n, err := vtString.Decode(dec, false)

		assert.Empty(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "decode int")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF before the value has been completely read", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 5)
		assert.NoError(t, err)

		enc.Buffer.Write(make([]byte, 2))

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		value, n, err := vtString.Decode(dec, false)

		assert.Empty(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "decode string")
	})

	t.Run("returns error from decoding value", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 5)
		assert.NoError(t, err)

		enc.Buffer.Write(make([]byte, 5))

		dec := NewDecoder(&errReadWriter{
			buf:   enc.Buffer,
			err:   io.ErrClosedPipe,
			errOn: 5,
		})

		value, n, err := vtString.Decode(dec, false)
		assert.Empty(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "decode string")
	})

	t.Run("returns ErrStringInvalid when decoded string is not valid UTF-8", func(t *testing.T) {
		t.Parallel()

		value := []byte{0xff, 0xfe, 0xfd}

		enc := NewEncoder()
		err := vtInt.Encode(enc, int64(len(value)))
		assert.NoError(t, err)

		enc.Buffer.Write(value)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actual, n, err := vtString.Decode(dec, false)

		assert.Empty(t, actual)
		assert.Equal(t, len(value)+1, n)
		assert.ErrorIs(t, err, ErrStringInvalid)
	})

	t.Run("returns empty string from decoder", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := vtInt.Encode(enc, 0)
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actual, n, err := vtString.Decode(dec, false)
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, "", actual)
	})

	t.Run("returns the decoded value", func(t *testing.T) {
		t.Parallel()

		value := "Cześć, こんにちは, 你好"
		enc := NewEncoder()

		err := vtInt.Encode(enc, int64(len(value)))
		assert.NoError(t, err)

		enc.Buffer.WriteString(value)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actual, n, err := vtString.Decode(dec, false)
		assert.NoError(t, err)

		assert.Equal(t, len(value)+1, n)
		assert.Equal(t, value, actual)
	})
}
