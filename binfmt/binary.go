package binfmt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var _ ValueType[[]byte] = Binary{}

// Binary implements [ValueType][[]byte] and is able to
// encode and decode []byte values. Keep in sync with
// type alias in kvega package.
type Binary struct{}

// Ident returns the Binary identifier.
func (Binary) Ident() byte {
	return 'B'
}

// Encode encodes value by adding its length to the encoders current buffer
// and then moves the current buffer to the writes list, followed by adding
// value to the writes list without a copy of value, it finally creates
// a new empty current buffer for further writes. value may contain any
// byte including NUL. Encode returns nil error.
func (Binary) Encode(enc *Encoder, value []byte) error {
	var vtInt Int

	// Encoding int does not return error.
	//nolint:errcheck
	//gosec:disable G104
	vtInt.Encode(enc, int64(len(value)))

	enc.writes = append(enc.writes, enc.Buffer, bytes.NewReader(value))
	enc.Buffer = new(bytes.Buffer)

	return nil
}

// Decode decodes arbitrary bytes from the decoders reader. First by reading the
// length using Int, followed by reading the actual bytes. If createCopy is false
// the returned byte slice is only valid until the next call made to the [Decoder].
func (Binary) Decode(dec *Decoder, createCopy bool) ([]byte, int, error) {
	var vtInt Int

	length, lenn, err := vtInt.Decode(dec, false)
	if err != nil {
		return nil, lenn, fmt.Errorf("binfmt: failed to decode binary length: %w", err)
	}

	if length == 0 {
		return dec.buffer[:0], lenn, nil
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
			return nil, lenn + n, fmt.Errorf("binfmt: failed to decode binary: %w", err)
		}
	}

	value := bytes
	if createCopy {
		value = make([]byte, len(bytes))
		copy(value, bytes)
	}

	return value, lenn + n, nil
}
