package binfmt

import (
	"bufio"
	"fmt"
)

const (
	// DeleteRowIdent is the type identifier for a delete row.
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

// ReadDeleteRow reads a delete row from reader, and expects that the row
// identifier byte has already been read to determine the kind of row
// to read.
func ReadDeleteRow(reader *bufio.Reader) (string, int, error) {
	key, keyn, err := ReadString(reader)
	if err != nil {
		return "", keyn, fmt.Errorf("binfmt: failed to read key: %w", err)
	}

	return key, keyn, nil
}
