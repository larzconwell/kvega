package binfmt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"slices"

	"golang.org/x/exp/constraints"
)

const byteSize = 7

var (
	// ErrReadUintInvalid is returned when a uint cannot be read due to invalid reader data.
	ErrReadUintInvalid = errors.New("binfmt: read uint failed due to invalid reader data")
	// ErrReadUintOverflow is returned when a read uint is larger than the given uint size.
	ErrReadUintOverflow = errors.New("binfmt: read uint does not fit in uint size")
)

// WriteUint writes a uint to writer using big-endian variable length integer encoding. The
// variable length integer encoding may write from 1 to 10 bytes depending on the value.
//
// Variable length integer encoding works by splitting the bits of a given value up into chunks
// of 7 bits, these chunks are structured as bytes where the least significant portion of the
// bytes contain the 7 bits from the chunks. For the bytes, the most significant bit is reserved
// and if set signifies that more bytes should be read, if unset signifies that the current byte
// is the last byte in the variable length integer data. WriteUint splits the value up and orders
// the resulting bytes in big-endian order.
//
// Here is an example of how a uint would be encoded and written to the writer. Given a value
// 0x108a, the following is a mapping between the values binary representation
// and the written bytes binary representation.
//
// [___10000] [10001010] - Binary representation of value.
// [__100001] [_0001010] - Spliting value into 7 bit chunks.
// [10100001] [00001010] - Adding continuation bits to the 7 bit chunks.
func WriteUint[T constraints.Unsigned](writer io.Writer, value T) (int, error) {
	// Determine number of bytes required to store encoded value.
	msbidx := bits.Len64(uint64(value))
	size := msbidx / byteSize

	if msbidx%byteSize > 0 {
		size++
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

	n, err := writer.Write(bytes)
	if err != nil {
		return n, fmt.Errorf("binfmt: failed to write uint: %w", err)
	}

	return n, nil
}

// ReadUint reads a big-endian variable length encoded uint from reader. It will read
// bytes until it detects the end of the encoded uint, up to 10 bytes. The read uint
// is parsed as a 64bit uint and returns ErrReadUintOverflow if the value is too large
// to fit into T.
func ReadUint[T constraints.Unsigned](reader *bufio.Reader) (T, int, error) {
	maxBytes := 10
	bytes := make([]byte, 0, maxBytes)

	// Read up to the max bytes expected for an encoded uint.
	for iter := range maxBytes {
		value, err := reader.ReadByte()
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}

		if err != nil {
			return 0, len(bytes), fmt.Errorf("binfmt: failed to read uint: %w", err)
		}

		bytes = append(bytes, value)

		if !continuationSet(value) {
			break
		}

		// If the continuation is not set and we've reached maxBytes we have an invalid stream.
		if iter >= maxBytes-1 {
			return 0, len(bytes), ErrReadUintInvalid
		}
	}

	var (
		value uint64
		shift uint
	)

	// Loop the bytes in little-endian order since we don't know how
	// many shifts are required to build the original value.
	for _, byt := range slices.Backward(bytes) {
		value |= uint64(setContinuation(byt, false)) << shift
		shift += 7
	}

	if value > uint64(T(0)-1) {
		return 0, len(bytes), ErrReadUintOverflow
	}

	return T(value), len(bytes), nil
}

func setContinuation(value byte, set bool) byte {
	if set {
		return value | 0x80
	}

	return value & 0x7f
}

func continuationSet(value byte) bool {
	return value&0x80 == 0x80
}
