package binfmt

import (
	"errors"
	"fmt"
	"io"
)

const (
	// SetRowIdent is the row identifier for a set row.
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

// SetRow reads a set row from the decoders reader, expecting that the row
// identifier has already been read to determine that the row is a set row.
// The returned value byte slice is only valid until the next call made to
// the Decoder. If the set row contains an invalid value type identifier,
// ErrRowValueIdentInvalid is returned.
func (dec *Decoder) SetRow() (string, []byte, int, error) {
	key, keyn, err := dec.String()
	if err != nil {
		return "", nil, keyn, fmt.Errorf("binfmt: failed to read key: %w", err)
	}

	valueIdent, err := dec.reader.ReadByte()
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return "", nil, keyn, fmt.Errorf("binfmt: failed to read value type identifier: %w", err)
	}

	switch valueIdent {
	case BinaryIdent:
		value, valuen, err := dec.Binary()
		if err != nil {
			return "", nil, keyn + 1 + valuen, fmt.Errorf("binfmt: failed to read binary value: %w", err)
		}

		return key, value, keyn + 1 + valuen, nil
	default:
		return "", nil, keyn + 1, ErrRowValueIdentInvalid
	}
}
