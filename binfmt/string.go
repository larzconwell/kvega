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

var _ ValueType[string] = String{}

// String implements [ValueType][string] and is able to
// encode and decode string values.
type String struct{}

// Ident returns the String identifier.
func (String) Ident() byte {
	return 'S'
}

// Encode encodes value by adding its length to the encoders current buffer
// and then moves the current buffer to the writes list, followed by adding
// value to the writes list, it finally creates a new empty current buffer
// for further writes. [ErrStringInvalid] is returned when value is not
// valid UTF-8.
func (String) Encode(enc *Encoder, value string) error {
	if !utf8.ValidString(value) {
		return ErrStringInvalid
	}

	var vtInt Int

	// Encoding int does not return error.
	//nolint:errcheck
	//gosec:disable G104
	vtInt.Encode(enc, int64(len(value)))

	enc.writes = append(enc.writes, enc.Buffer, strings.NewReader(value))
	enc.Buffer = new(bytes.Buffer)

	return nil
}

// Decode decodes a string from the decoders reader. First by reading the length
// using [Int.Decode], followed by reading the bytes for the string. [ErrStringInvalid]
// is returned when the bytes decoded contain invalid UTF-8 runes.
func (String) Decode(dec *Decoder) (string, int, error) {
	var vtInt Int

	length, lenn, err := vtInt.Decode(dec)
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
