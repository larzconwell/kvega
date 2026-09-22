package binfmt

import (
	"errors"
	"fmt"
	"io"
	"math/bits"
	"slices"
)

var (
	// ErrReadUintInvalid is returned when a uint cannot be read due to invalid reader data.
	ErrReadUintInvalid = errors.New("binfmt: read uint failed due to invalid reader data")
	// ErrReadUintOverflow is returned when a read uint is larger uint64.
	ErrReadUintOverflow = errors.New("binfmt: read uint does not fit in uint64")
)

// Uint encodes value using big-endian variable length integer encoding and adds it to the
// encoder. 1 to 10 bytes may be added to the encoder depending on the value.
//
// Variable length integer encoding works by splitting value into 7 bit chunks, where the
// 7 bits of value data are located in the least significant portion of a byte, the most
// significant bit in the chunk being reserved as a continuation bit to signal that more
// bytes should be read. Uint splits the value up and orders the resulting bytes in
// big-endian order.
//
// Here is an example of how a uint would be encoded. Given a value 0x108a, the following
// is a mapping between the values binary representation and the encoded bytes binary
// representation.
//
// [___10000] [10001010] - Binary representation of value 0x108a.
// [__100001] [_0001010] - Splitting value into 7 bit chunks.
// [10100001] [00001010] - Adding continuation bits to the 7 bit chunks.
func (enc *Encoder) Uint(value uint64) {
	// Determine number of bytes required to store encoded value.
	msbidx := bits.Len64(value)
	byteSize := 7
	size := msbidx / byteSize

	if msbidx%byteSize > 0 {
		size++
	}

	if size == 0 {
		size = 1
	}

	bytes := make([]byte, size)
	idx := size - 1
	last := true

	// Shift out from LSB to MSB 7 bit chunks at a time, writing
	// them to bytes in big-endian order and setting the continuation
	// bit depending on if it's the last byte or not.
	for value >= 0x80 {
		bytes[idx] = setContinuation(byte(value), !last)
		idx--

		value >>= byteSize
		last = false
	}

	bytes[idx] = setContinuation(byte(value), !last)
	enc.Buffer.Write(bytes)
}

// Uint reads a big-endian variable length integer encoded value from the decoders
// reader. It will read bytes until it detects the end of the encoded uint, up to
// 10 bytes. If the read value overflows uint64 ErrReadUintOverflow is returned.
func (dec *Decoder) Uint() (uint64, int, error) {
	var n int

	maxBytes := 10
	bytes := dec.getBuf(maxBytes)

	// Read up to the max bytes expected for an encoded uint.
	for iter := range maxBytes {
		value, err := dec.reader.ReadByte()
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}

		if err != nil {
			return 0, n, fmt.Errorf("binfmt: failed to read uint: %w", err)
		}

		bytes[iter] = value
		n++

		if !continuationSet(value) {
			break
		}

		// If the continuation is not set and we've reached maxBytes we have an invalid stream.
		if iter >= maxBytes-1 {
			return 0, n, ErrReadUintInvalid
		}
	}

	// Detect overflow, for math.MaxUint64 the max bytes would've been read
	// and the leading byte would only have the least significant bit and
	// the continuation bit set. If the leading byte has anything more than
	// the least significant bit set then the value would overflow uint64.
	if n >= maxBytes && bytes[0]&0x7e > 0 {
		return 0, n, ErrReadUintOverflow
	}

	var (
		value uint64
		shift uint
	)

	// Loop the bytes in little-endian order since we don't know how
	// many shifts are required to build the original value.
	for _, byt := range slices.Backward(bytes[:n]) {
		value |= uint64(setContinuation(byt, false)) << shift
		shift += 7
	}

	return value, n, nil
}

// setContinuation sets the most significant bit (the continuation
// bit) in value if set is true, or unsets it if set is false.
func setContinuation(value byte, set bool) byte {
	if set {
		return value | 0x80
	}

	return value & 0x7f
}

// continuationSet returns true if the most significant bit
// (the continuation bit) is set in value, false otherwise.
func continuationSet(value byte) bool {
	return value&0x80 == 0x80
}
