package binfmt

import (
	"bytes"
	"fmt"
	"io"
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

// WriteTo implements io.WriterTo and writes added row
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
