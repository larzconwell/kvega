package binfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

var (
	// ErrStringInvalid is returned when a string contains invalid UTF-8 runes.
	ErrStringInvalid = errors.New("binfmt: string contains invalid data")
)

// WriteString writes the length of the string in bytes to the writer
// using WriteUint and then the bytes of the string are written.
// ErrWriteStringInvalid is returned when value is not valid UTF-8.
func WriteString(writer *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return ErrStringInvalid
	}

	WriteInt(writer, int64(len(value)))
	writer.WriteString(value)

	return nil
}

// ReadString reads a string from reader, first by reading the length in
// bytes using ReadUint, followed by reading the bytes themselves.
// ErrReadStringInvalid is returned when the bytes read contain
// invalid UTF-8 runes.
func ReadString(reader *bufio.Reader) (string, int, error) {
	length, lenn, err := ReadInt[int64](reader)
	if err != nil {
		return "", lenn, fmt.Errorf("binfmt: failed to read string length: %w", err)
	}

	if length == 0 {
		return "", lenn, nil
	}

	var buf bytes.Buffer

	n, err := io.CopyN(&buf, reader, length)
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return "", lenn + int(n), fmt.Errorf("binfmt: failed to read string: %w", err)
	}

	bytes := buf.Bytes()
	if !utf8.Valid(bytes) {
		return "", lenn + int(n), ErrStringInvalid
	}

	return string(bytes), lenn + int(n), nil
}
