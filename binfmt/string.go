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
	// ErrStringInvalid is returned when a string contains invalid UTF-8 runes.
	ErrStringInvalid = errors.New("binfmt: string contains invalid data")
)

// String encodes value by adding its length to the encoders current buffer
// and then moves the current buffer to the writes list, followed by adding
// value to the writes list, it finally creates a new empty current buffer
// for further writes. ErrStringInvalid is returned when value is not
// valid UTF-8.
func (enc *Encoder) String(value string) error {
	if !utf8.ValidString(value) {
		return ErrStringInvalid
	}

	enc.Int(int64(len(value)))

	enc.writes = append(enc.writes, enc.Buffer, strings.NewReader(value))
	enc.Buffer = new(bytes.Buffer)

	return nil
}

// ReadString reads a string from reader, first by reading the length in
// bytes using ReadInt, followed by reading the bytes themselves.
// ErrStringInvalid is returned when the bytes read contain invalid
// UTF-8 runes.
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
