package binfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// WriteBinary writes the length of the slice to the writer using
// WriteUint and then the slice itself is written. The slice may
// contain any byte including NUL.
func WriteBinary(writer io.Writer, value []byte) (int, error) {
	// TODO: Write an int instead of a uint to ease reading, sizes won't really exceed int64 anyway.
	lenn, err := WriteUint(writer, uint64(len(value)))
	if err != nil {
		return lenn, err
	}

	reader := bytes.NewReader(value)

	n, err := io.Copy(writer, reader)
	if err != nil {
		return lenn + int(n), fmt.Errorf("binfmt: failed to write binary: %w", err)
	}

	return lenn + int(n), nil
}

// ReadBinary reads a slice from reader, first by reading the length
// using ReadUint, followed by reading the actual bytes themselves.
func ReadBinary(reader *bufio.Reader) ([]byte, int, error) {
	// TODO: Read an int instead of a uint to avoid unsafe conversion in io.CopyN.
	length, lenn, err := ReadUint[uint64](reader)
	if err != nil {
		return nil, lenn, err
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
		return nil, lenn + int(n), fmt.Errorf("binfmt: failed to read binary: %w", err)
	}

	return buf.Bytes(), lenn + int(n), nil
}
