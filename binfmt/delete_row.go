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

	var vtString = String{}

	err := vtString.Encode(enc, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	return nil
}

// DeleteRow decodes a delete row from the decoders reader, expecting
// that the row identifier has already been decoded to determine that
// the row is a delete row.
func (dec *Decoder) DeleteRow() (string, int, error) {
	var vtString = String{}

	key, keyn, err := vtString.Decode(dec)
	if err != nil {
		return "", keyn, fmt.Errorf("binfmt: failed to decode key: %w", err)
	}

	return key, keyn, nil
}
