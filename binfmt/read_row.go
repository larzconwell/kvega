package binfmt

import (
	"errors"
	"fmt"
)

var (
	// ErrRowIdentInvalid is returned when an invalid row identifier has been read.
	ErrRowIdentInvalid = errors.New("binfmt: row has invalid row identifier")
	// ErrRowValueIdentInvalid is returned when a row is read that has an invalid type identifier.
	ErrRowValueIdentInvalid = errors.New("binfmt: row has invalid value identifier")
)

// RowIdent reads a row identifier from the decoders reader and returns it.
// If the row identifier is invalid, ErrRowIdentInvalid is returned and the
// read byte is undread allowing for use later. io.EOF is returned if the
// end of the decoders reader has been reached.
func (dec *Decoder) RowIdent() (byte, error) {
	ident, err := dec.reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("binfmt: failed to read row identifier: %w", err)
	}

	switch ident {
	case SetRowIdent, DeleteRowIdent:
		return ident, nil
	default:
		// The invalid ident error is more important to surface.
		//nolint:errcheck
		//gosec:disable G104
		dec.reader.UnreadByte()

		return 0, ErrRowIdentInvalid
	}
}
