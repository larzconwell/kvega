package binfmt

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from WriteString", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := WriteSetRow(&buf, string([]byte{0xff, 0xfe, 0xfd}), []byte("value"))
		assert.ErrorIs(t, err, ErrWriteStringInvalid)
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
		assert.Equal(t, byte('S'), buf.Bytes()[0])
		assert.Equal(t, byte(0b0001_1110), buf.Bytes()[1])
		assert.Equal(t, []byte(key), buf.Bytes()[2:17])
		assert.Equal(t, byte('B'), buf.Bytes()[17])
		assert.Equal(t, byte(0b0000_0110), buf.Bytes()[18])
		assert.Equal(t, value, buf.Bytes()[19:])
	})
}
