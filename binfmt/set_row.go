package binfmt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const (
	// SetRowIdent is the type identifier for a set row.
	SetRowIdent = 'S'
)

// SetRow encodes a set row with key and value adding it to the encoder.
func (enc *Encoder) SetRow(key string, value []byte) error {
	enc.Buffer.WriteByte(SetRowIdent)

	err := enc.String(key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	enc.Buffer.WriteByte(BinaryIdent)
	enc.Binary(value)

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
