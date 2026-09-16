package binfmt

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from WriteString", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteDeleteRow(&buf, string([]byte{0xff, 0xfe, 0xfd}))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "write key")
	})

	t.Run("writes delete row to writer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		key := "こんにちは"

		err := WriteDeleteRow(&buf, key)
		assert.NoError(t, err)

		assert.Equal(t, 1+1+15, buf.Len())
		assert.Equal(t, byte(DeleteRowIdent), buf.Bytes()[0])
		assert.Equal(t, byte(0b0001_1110), buf.Bytes()[1])
		assert.Equal(t, []byte(key), buf.Bytes()[2:17])
	})
}
