package binfmt

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadRowIdent(t *testing.T) {
	t.Parallel()

	t.Run("returns error from reading byte", func(t *testing.T) {
		t.Parallel()

		ident, err := ReadRowIdent(bufio.NewReader(bytes.NewBuffer(nil)))
		assert.Zero(t, ident)
		assert.ErrorIs(t, err, io.EOF)
		assert.ErrorContains(t, err, "read row identifier")
	})

	t.Run("returns ErrRowIdentInvalid if read identifier is invalid and unreads the byte", func(t *testing.T) {
		t.Parallel()

		buf := bytes.NewBuffer([]byte{'Z'})
		reader := bufio.NewReader(buf)

		ident, err := ReadRowIdent(reader)
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

		ident, err := ReadRowIdent(reader)
		assert.NoError(t, err)

		assert.Equal(t, byte(SetRowIdent), ident)

		_, err = reader.ReadByte()
		assert.ErrorIs(t, err, io.EOF)
	})
}
