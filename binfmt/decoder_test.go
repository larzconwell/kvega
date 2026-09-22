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
