package binfmt

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

// ErrRowIdentInvalid is returned when an invalid row identifier has been decoded.
var ErrRowIdentInvalid = errors.New("binfmt: row has invalid row identifier")

// Decoder is used to efficiently read and decode row
// data from a reader.
type Decoder struct {
	reader *bufio.Reader
	buffer []byte
}

// NewDecoder creates a decoder that reads from reader.
func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{
		reader: bufio.NewReader(reader),
		buffer: make([]byte, bytes.MinRead),
	}
}

// Row decodes a row identifier and key from the decodes reader and returns it
// along with a row value decoder if the row has a value associated with it.
// If the row identifier is invalid, [ErrRowIdentInvalid] is returned. [io.EOF]
// is returned if the end of the decoders reader has been reached.
func (dec *Decoder) Row() (byte, string, RowValueDecoder, int, error) {
	var (
		vtString String
		rvd      RowValueDecoder
	)

	ident, err := dec.reader.ReadByte()
	if err != nil {
		return 0, "", rvd, 0, fmt.Errorf("binfmt: failed to decode row identifier: %w", err)
	}

	switch ident {
	case SetRowIdent:
		key, keyn, err := vtString.Decode(dec, false)
		if err != nil {
			return 0, "", rvd, 1 + keyn, fmt.Errorf("binfmt: failed to decode key: %w", err)
		}

		return ident, key, RowValueDecoder{HasValue: true, dec: dec}, 1 + keyn, nil
	case DeleteRowIdent:
		key, keyn, err := vtString.Decode(dec, false)
		if err != nil {
			return 0, "", rvd, 1 + keyn, fmt.Errorf("binfmt: failed to decode key: %w", err)
		}

		return ident, key, RowValueDecoder{}, 1 + keyn, nil
	default:
		return 0, "", rvd, 1, ErrRowIdentInvalid
	}
}

// getBuf returns a byte slice of length n, where the backing
// slice is shared across the entire decoder and is grown as
// needed by calls to getBuf.
func (dec *Decoder) getBuf(n int) []byte {
	if len(dec.buffer) >= n {
		return dec.buffer[:n]
	}

	// Grow buffers length to n, and reslice so that buffer
	// encompasses the full capcity allocated by append. This
	// should increase the chance of hitting the short circuit.
	dec.buffer = append(dec.buffer, make([]byte, n-len(dec.buffer))...)
	dec.buffer = dec.buffer[:cap(dec.buffer)]

	return dec.buffer[:n]
}
