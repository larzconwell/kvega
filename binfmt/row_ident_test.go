package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecoderRowIdent(t *testing.T) {
	t.Parallel()

	t.Run("returns error from decoding byte", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(bytes.NewBuffer(nil))
		ident, err := dec.RowIdent()

		assert.Zero(t, ident)
		assert.ErrorIs(t, err, io.EOF)
		assert.ErrorContains(t, err, "decode row identifier")
	})

	t.Run("returns ErrRowIdentInvalid if decoded identifier is invalid and unreads the byte", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBuffer([]byte{'Z'})
		reader := bufio.NewReader(buf)

		dec := NewDecoder(reader)
		ident, err := dec.RowIdent()
		assert.Zero(t, ident)
		assert.ErrorIs(t, err, ErrRowIdentInvalid)

		byt, err := reader.ReadByte()
		assert.NoError(t, err)
		assert.Equal(t, byte('Z'), byt)
	})

	t.Run("returns row identifier if valid", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBuffer([]byte{SetRowIdent})
		reader := bufio.NewReader(buf)

		dec := NewDecoder(reader)
		ident, err := dec.RowIdent()
		assert.NoError(t, err)

		assert.Equal(t, byte(SetRowIdent), ident)

		_, err = reader.ReadByte()
		assert.ErrorIs(t, err, io.EOF)
	})
}
