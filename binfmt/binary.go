package binfmt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	// BinaryIdent is the type identifier for a binary value.
	BinaryIdent = 'B'
)

// Binary encodes value by adding its length to the encoders current buffer
// and then moves the current buffer to the writes list, followed by adding
// value to the writes list without a copy of value, it finally creates
// a new empty current buffer for further writes. value may contain any
// byte including NUL.
func (enc *Encoder) Binary(value []byte) {
	enc.Int(int64(len(value)))

	enc.writes = append(enc.writes, enc.Buffer, bytes.NewReader(value))
	enc.Buffer = new(bytes.Buffer)
}

// Binary reads arbitrary bytes from the decoders reader. First by reading the
// length using Int, followed by reading the actual bytes. The returned byte
// slice is only valid until the next call made to the Decoder.
func (dec *Decoder) Binary() ([]byte, int, error) {
	length, lenn, err := dec.Int()
	if err != nil {
		return nil, lenn, fmt.Errorf("binfmt: failed to read binary length: %w", err)
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
			return nil, lenn + n, fmt.Errorf("binfmt: failed to read binary: %w", err)
		}
	}

	return bytes, lenn + n, nil
}
