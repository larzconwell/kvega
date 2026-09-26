package binfmt

import (
	"bytes"
	"fmt"
	"io"
)

const (
	// SetRowIdent is the row identifier for a set row.
	SetRowIdent = 'S'
	// DeleteRowIdent is the row identifier for a delete row.
	DeleteRowIdent = 'D'
)

// Encoder is used to efficiently encode and write row data
// with minimal copying.
type Encoder struct {
	// Buffer contains the current buffer that may be written to.
	Buffer *bytes.Buffer
	writes []io.WriterTo
}

// NewEncoder creates an empty encoder.
func NewEncoder() *Encoder {
	return &Encoder{Buffer: new(bytes.Buffer)}
}

// WriteTo implements [io.WriterTo] and writes added row
// data to w.
func (enc *Encoder) WriteTo(w io.Writer) (int64, error) {
	var total int64

	for _, write := range enc.writes {
		n, err := write.WriteTo(w)
		total += n

		if err != nil {
			return total, fmt.Errorf("binfmt: failed to write encoded row data: %w", err)
		}
	}

	if enc.Buffer.Len() > 0 {
		n, err := enc.Buffer.WriteTo(w)
		total += n

		if err != nil {
			return total, fmt.Errorf("binfmt: failed to write encoded row data: %w", err)
		}
	}

	return total, nil
}

// SetRow encodes a set row with key and value adding it to the encoder.
func (enc *Encoder) SetRow[VT ValueType[T], T any](key string, value T) error {
	enc.Buffer.WriteByte(SetRowIdent)

	var vtString String

	err := vtString.Encode(enc, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	var vt VT
	enc.Buffer.WriteByte(vt.Ident())

	err = vt.Encode(enc, value)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode value: %w", err)
	}

	return nil
}

// DeleteRow encodes a delete row with key adding it to the encoder.
func (enc *Encoder) DeleteRow(key string) error {
	enc.Buffer.WriteByte(DeleteRowIdent)

	var vtString String

	err := vtString.Encode(enc, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	return nil
}
