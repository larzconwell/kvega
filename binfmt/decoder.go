package binfmt

import (
	"bufio"
	"bytes"
	"io"
)

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
