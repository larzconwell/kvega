package binfmt

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEncoder(t *testing.T) {
	t.Parallel()

	enc := NewEncoder()
	assert.NotNil(t, enc.Buffer)
	assert.Empty(t, enc.writes)
}

func TestEncoderWriteTo(t *testing.T) {
	t.Parallel()

	t.Run("returns number of bytes written and error when writing a write", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.writes = append(enc.writes, strings.NewReader("1,"))
		enc.writes = append(enc.writes, strings.NewReader("2,"))
		enc.writes = append(enc.writes, strings.NewReader("3"))

		var buf bytes.Buffer

		n, err := enc.WriteTo(&errReadWriter{buf: &buf, errOn: 3, err: io.ErrClosedPipe})
		assert.Equal(t, int64(3), n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write encoded row data")

		assert.Equal(t, "1,2", buf.String())
	})

	t.Run("returns number of bytes written and error when writing current buffer", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.writes = append(enc.writes, strings.NewReader("1,"))
		enc.Buffer.WriteString("2,3")

		var buf bytes.Buffer

		n, err := enc.WriteTo(&errReadWriter{buf: &buf, errOn: 3, err: io.ErrClosedPipe})
		assert.Equal(t, int64(3), n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "write encoded row data")

		assert.Equal(t, "1,2", buf.String())
	})

	t.Run("returns number of bytes written and writes in expected order", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.writes = append(enc.writes, strings.NewReader("1,"))
		enc.writes = append(enc.writes, strings.NewReader("2,"))
		enc.writes = append(enc.writes, strings.NewReader("3,"))
		enc.Buffer.WriteString("4")

		var buf bytes.Buffer

		n, err := enc.WriteTo(&buf)
		assert.NoError(t, err)

		assert.Equal(t, int64(7), n)
		assert.Equal(t, "1,2,3,4", buf.String())
	})
}
