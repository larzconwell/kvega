package binfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

const (
	// SetRowIdent is the type identifier for a set row.
	SetRowIdent = 'S'
)

// WriteSetRow writes a set row with the given key and value.
func WriteSetRow(writer *bytes.Buffer, key string, value []byte) error {
	writer.WriteByte(SetRowIdent)

	err := WriteString(writer, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to write key: %w", err)
	}

	writer.WriteByte('B')
	WriteBinary(writer, value)

	return nil
}

// ReadSetRow reads a set row from reader, and expects that the row
// identifier byte has already been read to determine the kind of row
// to read. If the set row contains an invalid type,
// ErrRowValueIdentInvalid is returned.
func ReadSetRow(reader *bufio.Reader) (string, []byte, int, error) {
	key, keyn, err := ReadString(reader)
	if err != nil {
		return "", nil, keyn, fmt.Errorf("binfmt: failed to read key: %w", err)
	}

	valueIdent, err := reader.ReadByte()
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return "", nil, keyn, fmt.Errorf("binfmt: failed to read value identifier: %w", err)
	}

	switch valueIdent {
	case BinaryIdent:
		value, valuen, err := ReadBinary(reader)
		if err != nil {
			return "", nil, keyn + 1 + valuen, fmt.Errorf("binfmt: failed to read binary value: %w", err)
		}

		return key, value, keyn + 1 + valuen, nil
	default:
		return "", nil, keyn + 1, ErrRowValueIdentInvalid
	}
}
