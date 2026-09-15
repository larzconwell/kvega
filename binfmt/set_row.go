package binfmt

import (
	"bytes"
	"fmt"
)

// WriteSetRow writes a set row with the given key and value.
func WriteSetRow(writer *bytes.Buffer, key string, value []byte) error {
	writer.WriteByte('S')

	err := WriteString(writer, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to write key: %w", err)
	}

	writer.WriteByte('B')
	WriteBinary(writer, value)

	return nil
}
