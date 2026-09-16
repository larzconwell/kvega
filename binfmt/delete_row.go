package binfmt

import (
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
