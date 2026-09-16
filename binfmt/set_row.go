package binfmt

import (
	"bytes"
	"fmt"
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
