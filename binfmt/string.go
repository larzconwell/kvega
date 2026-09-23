package binfmt

import (
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

// String decodes a string from the decoders reader. First by reading the length
// using Int, followed by reading the bytes for the string. ErrStringInvalid
// is returned when the bytes decoded contain invalid UTF-8 runes.
func (dec *Decoder) String() (string, int, error) {
	length, lenn, err := dec.Int()
	if err != nil {
		return "", lenn, fmt.Errorf("binfmt: failed to decode string length: %w", err)
	}

	if length == 0 {
		return "", lenn, nil
	}

	bytes := dec.getBuf(int(length))

	var n int
	for n < len(bytes) {
		nn, err := dec.reader.Read(bytes[n:])
		n += nn

		if errors.Is(err, io.EOF) {
			if n >= len(bytes) {
				break
			}

			err = io.ErrUnexpectedEOF
		}

		if err != nil {
			return "", lenn + n, fmt.Errorf("binfmt: failed to decode string: %w", err)
		}
	}

	if !utf8.Valid(bytes) {
		return "", lenn + n, ErrStringInvalid
	}

	return string(bytes), lenn + n, nil
}
