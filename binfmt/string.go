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

	// TODO: Write an int instead of a uint to ease reading, sizes won't really exceed int64 anyway.
	lenn, err := WriteUint(writer, uint64(len(value)))
	if err != nil {
		return lenn, err
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
	// TODO: Read an int instead of a uint to avoid unsafe conversion in io.CopyN.
	length, lenn, err := ReadUint[uint64](reader)
	if err != nil {
		return "", lenn, err
	}

	var buf bytes.Buffer

	// If length overflows int64 it'll wrap around to negative, and io.CopyN will
	// return 0, io.EOF. For now this will do until int reading is added.
	//gosec:disable G115
	n, err := io.CopyN(&buf, reader, int64(length))
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
