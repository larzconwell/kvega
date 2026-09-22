package binfmt

import (
	"fmt"
)

const (
	// DeleteRowIdent is the row identifier for a delete row.
	DeleteRowIdent = 'D'
)

// DeleteRow encodes a delete row with key adding it to the encoder.
func (enc *Encoder) DeleteRow(key string) error {
	enc.Buffer.WriteByte(DeleteRowIdent)

	err := enc.String(key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	return nil
}

// DeleteRow reads a delete row from the decoders reader, expecting
// that the row identifier has already been read to determine that
// the row is a delete row.
func (dec *Decoder) DeleteRow() (string, int, error) {
	key, keyn, err := dec.String()
	if err != nil {
		return "", keyn, fmt.Errorf("binfmt: failed to read key: %w", err)
	}

	return key, keyn, nil
}
