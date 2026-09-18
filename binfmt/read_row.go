package binfmt

import (
	"bufio"
	"errors"
	"fmt"
)

var (
	// ErrRowIdentInvalid is returned when an invalid row identifier has been read.
	ErrRowIdentInvalid = errors.New("binfmt: row has invalid row identifier")
	// ErrRowValueIdentInvalid is returned when a row is read that has an invalid type identifier.
	ErrRowValueIdentInvalid = errors.New("binfmt: row has invalid value identifier")
)

// ReadRowIdent reads a row identifier from the reader and returns it if it's
// valid. If the row identifier is invalid ErrRowIdentInvalid is returned and
// the read byte is unread causing this error to cycle.
func ReadRowIdent(reader *bufio.Reader) (byte, error) {
	ident, err := reader.ReadByte()
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
		reader.UnreadByte()

		return 0, ErrRowIdentInvalid
	}
}
