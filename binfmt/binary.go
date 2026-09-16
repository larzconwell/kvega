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

// WriteBinary writes the length of the slice to the writer using
// WriteUint and then the slice itself is written. The slice may
// contain any byte including NUL.
func WriteBinary(writer *bytes.Buffer, value []byte) {
	WriteInt(writer, int64(len(value)))
	writer.Write(value)
}

// ReadBinary reads a slice from reader, first by reading the length
// using ReadUint, followed by reading the actual bytes themselves.
func ReadBinary(reader *bufio.Reader) ([]byte, int, error) {
	length, lenn, err := ReadInt[int64](reader)
	if err != nil {
		return nil, lenn, fmt.Errorf("binfmt: failed to read binary length: %w", err)
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
