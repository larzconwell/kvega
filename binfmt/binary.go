package binfmt

import (
	"bufio"
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

// ReadBinary reads a slice from reader, first by reading the length
// using ReadInt, followed by reading the actual bytes themselves.
func ReadBinary(reader *bufio.Reader) ([]byte, int, error) {
	length, lenn, err := ReadInt[int64](reader)
	if err != nil {
		return nil, lenn, fmt.Errorf("binfmt: failed to read binary length: %w", err)
	}

	if length == 0 {
		return make([]byte, 0), lenn, nil
	}

	var buf bytes.Buffer

	n, err := io.CopyN(&buf, reader, length)
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return nil, lenn + int(n), fmt.Errorf("binfmt: failed to read binary: %w", err)
	}

	return buf.Bytes(), lenn + int(n), nil
}
