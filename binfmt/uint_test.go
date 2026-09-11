package binfmt

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

type errReadWriter struct {
	buf      *bytes.Buffer
	n        int
	errAfter int
	err      error
}

func (erw *errReadWriter) Read(p []byte) (int, error) {
	if erw.buf == nil {
		erw.buf = bytes.NewBuffer(nil)
	}

	maxRead := min(len(p), erw.errAfter-erw.n)

	n, err := erw.buf.Read(p[:maxRead])
	if err != nil {
		return n, fmt.Errorf("binfmt: failed to read: %w", err)
	}

	erw.n += n

	if erw.n >= erw.errAfter {
		return n, erw.err
	}

	return n, nil
}

func (erw *errReadWriter) Write(p []byte) (int, error) {
	if erw.buf == nil {
		erw.buf = bytes.NewBuffer(nil)
	}

	maxWrite := min(len(p), erw.errAfter-erw.n)

	n, err := erw.buf.Write(p[:maxWrite])
	if err != nil {
		return n, fmt.Errorf("binfmt: failed to write: %w", err)
	}

	erw.n += n

	if erw.n >= erw.errAfter {
		return n, erw.err
	}

	return n, nil
}

func TestWriteUint(t *testing.T) {
	t.Parallel()

	t.Run("returns writer error", func(t *testing.T) {
		t.Parallel()

		n, err := WriteUint(&errReadWriter{err: io.ErrClosedPipe}, uint(5))
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("writes small uint in one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		n, err := WriteUint(&buf, uint(127))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, byte(0b0111_1111), buf.Bytes()[0])
	})

	t.Run("writes medium uint in three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		n, err := WriteUint(&buf, uint(0xbbbb))
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, []byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		}, buf.Bytes())
	})

	t.Run("writes large uint in nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		n, err := WriteUint(&buf, uint64(0xbbb_bbbb_bbbb_bbbb))
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, []byte{
			0b1000_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b0011_1011,
		}, buf.Bytes())
	})

	t.Run("writes max uint in ten bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		n, err := WriteUint(&buf, uint64(math.MaxUint64))
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, []byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b0111_1111,
		}, buf.Bytes())
	})
}

func TestReadUint(t *testing.T) {
	t.Parallel()

	t.Run("return io.ErrUnexpectedEOF if encountering io.EOF before the last byte is read", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			// Expecting at least one more byte without continuation set.
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 3, n)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("return error if encountered error reading bytes", func(t *testing.T) {
		t.Parallel()

		value, n, err := ReadUint[uint64](bufio.NewReader(&errReadWriter{err: io.ErrClosedPipe}))
		assert.Zero(t, value)
		assert.Zero(t, n)
		assert.ErrorIs(t, err, io.ErrClosedPipe)
	})

	t.Run("return error if reached ten byte cap without encounting unset continuation bit", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff, // Should be 0b0111_1111 instead
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 10, n)
		assert.ErrorIs(t, err, ErrReadUintInvalid)
	})

	t.Run("return error if read uint is larger than fits in given uint type", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint8](bufio.NewReader(&buf))
		assert.Zero(t, value)
		assert.Equal(t, 2, n)
		assert.ErrorIs(t, err, ErrReadUintOverflow)
	})

	t.Run("reads small uint encoded as one byte", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.WriteByte(0b0111_1111)

		value, n, err := ReadUint[uint](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 1, n)
		assert.Equal(t, uint(127), value)
	})

	t.Run("reads medium uint encoded as three bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0010,
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 3, n)
		assert.Equal(t, uint(0xbbbb), value)
	})

	t.Run("reads large uint encoded as nine bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b1011_1011,
			0b1101_1101,
			0b1110_1110,
			0b1111_0111,
			0b0011_1011,
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 9, n)
		assert.Equal(t, uint64(0xbbb_bbbb_bbbb_bbbb), value)
	})

	t.Run("reads marg uint encoded as 10 bytes", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		buf.Write([]byte{
			0b1000_0001,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0xff,
			0b0111_1111,
		})

		value, n, err := ReadUint[uint64](bufio.NewReader(&buf))
		assert.NoError(t, err)

		assert.Equal(t, 10, n)
		assert.Equal(t, uint64(math.MaxUint64), value)
	})
}
