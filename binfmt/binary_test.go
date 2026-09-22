package binfmt

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoderBinary(t *testing.T) {
	t.Parallel()

	t.Run("encodes empty binary", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Binary(nil)

		assert.Len(t, enc.writes, 2)
		assert.NotNil(t, enc.Buffer)
		assert.NotSame(t, enc.writes[0], enc.Buffer)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{0}, enc.writes[0].(*bytes.Buffer).Bytes())
		//nolint:forcetypeassert
		assert.Zero(t, enc.writes[1].(*bytes.Reader).Size())
	})

	t.Run("encodes length and value", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		enc := NewEncoder()
		enc.Binary(value)

		assert.Len(t, enc.writes, 2)
		assert.NotNil(t, enc.Buffer)
		assert.NotSame(t, enc.writes[0], enc.Buffer)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			0b1000_0010,
			0,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*bytes.Reader))
		assert.NoError(t, err)
		assert.Equal(t, value, buf)
	})
}

func TestDecoderBinary(t *testing.T) {
	t.Parallel()

	t.Run("returns error from Int", func(t *testing.T) {
		t.Parallel()

		dec := NewDecoder(&errReadWriter{err: io.ErrClosedPipe})
		value, n, err := dec.Binary()

		assert.Empty(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read int")
	})

	t.Run("returns io.ErrUnexpectedEOF if encountered io.EOF before the value has been completely read", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Int(5)
		enc.Buffer.Write(make([]byte, 2))

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		value, n, err := dec.Binary()

		assert.Empty(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		assert.ErrorContains(t, err, "read binary")
	})

	t.Run("returns error from reading value", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Int(5)
		enc.Buffer.Write(make([]byte, 5))

		dec := NewDecoder(&errReadWriter{
			buf:   enc.Buffer,
			err:   io.ErrClosedPipe,
			errOn: 5,
		})

		value, n, err := dec.Binary()
		assert.Empty(t, value)
		assert.Equal(t, 5, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		assert.ErrorContains(t, err, "read binary")
	})

	t.Run("returns empty binary from reader", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		enc.Int(0)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actual, n, err := dec.Binary()
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, make([]byte, 0), actual)
	})

	t.Run("returns the read value", func(t *testing.T) {
		t.Parallel()

		value := make([]byte, 128)
		for idx := range value {
			value[idx] = byte(idx) % 127
		}

		enc := NewEncoder()
		enc.Int(int64(len(value)))
		enc.Buffer.Write(value)

		reader, err := encoderToBufReader(enc)
		assert.NoError(t, err)

		dec := NewDecoder(reader)
		actual, n, err := dec.Binary()
		assert.NoError(t, err)

		assert.Equal(t, len(value)+2, n)
		assert.Equal(t, value, actual)
	})
}
