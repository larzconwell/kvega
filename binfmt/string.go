package binfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

var (
	// ErrWriteStringInvalid is returned when a string contains invalid UTF-8 runes.
	ErrWriteStringInvalid = errors.New("binfmt: string contains invalid data")
	// ErrReadStringInvalid is returned when a read string contains invalid UTF-8 runes.
	ErrReadStringInvalid = errors.New("binfmt: read string contains invalid data")
)

// WriteString writes the length of the string in bytes to the writer
// using WriteUint and then the bytes of the string are written.
// ErrWriteStringInvalid is returned when value is not valid UTF-8.
func WriteString(writer io.Writer, value string) (int, error) {
	if !utf8.ValidString(value) {
		return 0, ErrWriteStringInvalid
	}

	lenn, err := WriteInt(writer, int64(len(value)))
	if err != nil {
		return lenn, fmt.Errorf("binfmt: failed to write string length: %w", err)
	}

	reader := strings.NewReader(value)

	n, err := io.Copy(writer, reader)
	if err != nil {
		return lenn + int(n), fmt.Errorf("binfmt: failed to write string: %w", err)
	}

	return lenn + int(n), nil
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
		return "", lenn + int(n), ErrReadStringInvalid
	}

	return string(bytes), lenn + int(n), nil
}
