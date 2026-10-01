package binfmt

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDecoder(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	buf.WriteString("test")

	dec := NewDecoder(&buf)
	assert.NotNil(t, dec.reader)
	assert.Len(t, dec.buffer, bytes.MinRead)

	data, err := io.ReadAll(dec.reader)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), data)
}

func TestDecoderRow(t *testing.T) {
	t.Parallel()

	t.Run("returns io.EOF if decoder is empty", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(bytes.NewBuffer(nil))
		ident, key, valueDecoder, n, err := dec.Row()

		assert.Zero(t, ident)
		assert.Zero(t, key)
		assert.Zero(t, valueDecoder)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.EOF)
		assert.ErrorContains(t, err, "decode row identifier")
	})

	t.Run("returns ErrRowIdentInvalid if decoded row identifier is invalid and unreads the row identifier", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBuffer([]byte{'Z'})
		dec := NewDecoder(buf)
		ident, key, valueDecoder, n, err := dec.Row()

		assert.Zero(t, ident)
		assert.Zero(t, key)
		assert.Zero(t, valueDecoder)
		assert.Equal(t, 1, n)
		assert.ErrorIs(t, err, ErrRowIdentInvalid)

		ident, err = dec.reader.ReadByte()
		assert.NoError(t, err)

		assert.Equal(t, byte('Z'), ident)
	})

	t.Run("returns error from decoding set row key", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(SetRowIdent)

		// Force error by writing invalid key string length.
		for range 10 {
			buf.WriteByte(0xff)
		}

		dec := NewDecoder(&buf)
		ident, key, valueDecoder, n, err := dec.Row()

		assert.Zero(t, ident)
		assert.Zero(t, key)
		assert.Zero(t, valueDecoder)
		assert.Equal(t, 1+10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode key")
	})

	t.Run("returns error from decoding delete row key", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(DeleteRowIdent)

		// Force error by writing invalid key string length.
		for range 10 {
			buf.WriteByte(0xff)
		}

		dec := NewDecoder(&buf)
		ident, key, valueDecoder, n, err := dec.Row()

		assert.Zero(t, ident)
		assert.Zero(t, key)
		assert.Zero(t, valueDecoder)
		assert.Equal(t, 1+10, n)
		assert.ErrorIs(t, err, ErrDecodeIntInvalid)
		assert.ErrorContains(t, err, "decode key")
	})

	t.Run("returns row identifier with key and value decoder if retrieved a set row", func(t *testing.T) {
		t.Parallel()

		var vtString String

		enc := NewEncoder()
		enc.Buffer.WriteByte(SetRowIdent)

		err := vtString.Encode(enc, "key")
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		ident, key, valueDecoder, n, err := dec.Row()
		assert.NoError(t, err)

		assert.Equal(t, byte(SetRowIdent), ident)
		assert.Equal(t, "key", key)
		assert.True(t, valueDecoder.HasValue)
		assert.Same(t, dec, valueDecoder.dec)
		assert.Equal(t, 1+1+3, n)
	})

	t.Run("returns row identifier with key and empty value decoder if retrieved a delete row", func(t *testing.T) {
		t.Parallel()

		var vtString String

		enc := NewEncoder()
		enc.Buffer.WriteByte(DeleteRowIdent)

		err := vtString.Encode(enc, "key")
		assert.NoError(t, err)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		ident, key, valueDecoder, n, err := dec.Row()
		assert.NoError(t, err)

		assert.Equal(t, byte(DeleteRowIdent), ident)
		assert.Equal(t, "key", key)
		assert.Zero(t, valueDecoder)
		assert.Equal(t, 1+1+3, n)
	})
}

func TestDecoderGetBuf(t *testing.T) {
	t.Parallel()

	// Specify the starting buffer so code can change initial size without breaking tests.
	dec := NewDecoder(bytes.NewBuffer(nil))
	dec.buffer = make([]byte, 5)

	buf := dec.getBuf(2)
	assert.Len(t, buf, 2)
	assert.Len(t, dec.buffer, 5)
	assert.Equal(t, 5, cap(dec.buffer))

	buf = dec.getBuf(5)
	assert.Len(t, buf, 5)
	assert.Len(t, dec.buffer, 5)
	assert.Equal(t, 5, cap(dec.buffer))

	buf = dec.getBuf(10)
	assert.Len(t, buf, 10)
	assert.GreaterOrEqual(t, len(dec.buffer), 10)
	assert.Len(t, dec.buffer, cap(dec.buffer))
}
