package binfmt

import (
	"bufio"
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/exp/constraints"
)

var (
	// ErrReadIntInvalid is returned when an int cannot be read due to invalid reader data.
	ErrReadIntInvalid = errors.New("binfmt: read int failed due to invalid reader data")
	// ErrReadIntOverflow is returned when a read int is larger than the given int size.
	ErrReadIntOverflow = errors.New("binfmt: read int does not fit in int size")
)

// Int encodes value using zigzag encoding along with big-endian variable length integer
// encoding and adds it to the encoder. 1 to 10 bytes may be added to the encoder
// depending on the value.
//
// The first step is zigzag encoding which is a method to represent a signed integer
// as an unsigned integer by evenly spreading positive and negative values across the
// unsigned integer range. Negative values become double their absolute value minus
// one. Positive values meanwhile become double their absolute value.
//
// After the int64 has been converted to a uint64, big-endian variable length
// inger encoding is done, context to which can be gathered by reading Uint.
func (enc *Encoder) Int(value int64) {
	bitSize := unsafe.Sizeof(value) * 8

	// This overflow is fine, tests validate the behavior when
	// encoding math.MinInt64 and math.MaxInt64.
	//gosec:disable G115
	uValue := (uint64(value) << 1) ^ uint64(value>>(bitSize-1))
	enc.Uint(uValue)
}

// ReadInt reads a big-endian variable length encoded uint that contains a
// zigzag encoded int. Once the uint has been read it is zigzag decoded to
// convert it back to the appropriate int value. Up to 10 bytes are read
// from reader.
func ReadInt[T constraints.Signed](reader *bufio.Reader) (T, int, error) {
	// Use uint64 since we have no clean way in Go to get an unsigned
	// version of T without adding more type parameters.
	uValue, n, err := ReadUint[uint64](reader)
	if errors.Is(err, ErrReadUintInvalid) {
		return 0, n, ErrReadIntInvalid
	}

	if err != nil {
		return 0, n, fmt.Errorf("binfmt: failed to read int: %w", errors.Unwrap(err))
	}

	// This theoretical overflow is fine, the conversion will lead
	// to sign extension bits being truncated but the calculation
	// ensures the value will be within the signed range.
	//gosec:disable G115
	value := int64((uValue >> 1) ^ (-(uValue & 1)))
	bitSize := unsafe.Sizeof(T(0)) * 8
	maxT := T((1 << (bitSize - 1)) - 1)

	if value > int64(maxT) {
		return 0, n, ErrReadIntOverflow
	}

	return T(value), n, err
}
