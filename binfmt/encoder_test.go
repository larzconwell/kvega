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

func TestEncoderSetRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from encoding key", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.SetRow[Binary](string([]byte{0xff, 0xfe, 0xfd}), []byte("value"))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "encode key")
	})

	t.Run("encodes set row key and uint value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := uint64(30)
		enc := NewEncoder()

		err := enc.SetRow[Uint](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtUint Uint

		assert.Equal(t, []byte{
			vtUint.Ident(),
			0b0001_1110,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes set row key and int value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := int64(30)
		enc := NewEncoder()

		err := enc.SetRow[Int](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtInt Int

		assert.Equal(t, []byte{
			vtInt.Ident(),
			0b0011_1100,
		}, enc.Buffer.Bytes())
	})

	t.Run("encodes set row key and binary value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		value := []byte{0xff, 0xfe, 0xfd}
		enc := NewEncoder()

		err := enc.SetRow[Binary](key, value)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 4)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtBinary Binary

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			vtBinary.Ident(),
			0b0000_0110,
		}, enc.writes[2].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err = io.ReadAll(enc.writes[3].(*bytes.Reader))
		assert.NoError(t, err)
		assert.Equal(t, value, buf)
	})

	t.Run("encodes set row key and string value", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := enc.SetRow[String](key, key)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 4)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			SetRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)

		var vtString String

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			vtString.Ident(),
			0b0001_1110,
		}, enc.writes[2].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err = io.ReadAll(enc.writes[3].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, key, string(buf))
	})
}

func TestEncoderDeleteRow(t *testing.T) {
	t.Parallel()

	t.Run("returns error from encoding key", func(t *testing.T) {
		t.Parallel()

		enc := NewEncoder()
		err := enc.DeleteRow(string([]byte{0xff, 0xfe, 0xfd}))
		assert.ErrorIs(t, err, ErrStringInvalid)
		assert.ErrorContains(t, err, "encode key")
	})

	t.Run("encodes delete rows key", func(t *testing.T) {
		t.Parallel()

		key := "こんにちは"
		enc := NewEncoder()

		err := enc.DeleteRow(key)
		assert.NoError(t, err)

		assert.Len(t, enc.writes, 2)

		//nolint:forcetypeassert
		assert.Equal(t, []byte{
			DeleteRowIdent,
			0b0001_1110,
		}, enc.writes[0].(*bytes.Buffer).Bytes())

		//nolint:forcetypeassert
		buf, err := io.ReadAll(enc.writes[1].(*strings.Reader))
		assert.NoError(t, err)
		assert.Equal(t, []byte(key), buf)
	})
}
