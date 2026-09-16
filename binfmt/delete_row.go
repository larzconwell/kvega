package binfmt

import (
	"bufio"
	"bytes"
	"fmt"
)

const (
	// DeleteRowIdent is the type identifier for a delete row.
	DeleteRowIdent = 'D'
)

// WriteDeleteRow writes a delete row for the given key.
func WriteDeleteRow(writer *bytes.Buffer, key string) error {
	writer.WriteByte(DeleteRowIdent)

	err := WriteString(writer, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to write key: %w", err)
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
