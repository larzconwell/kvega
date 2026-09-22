package binfmt

import (
	"errors"
	"fmt"
	"unsafe"
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
// integer encoding is done, context to which can be gathered by reading Uint.
func (enc *Encoder) Int(value int64) {
	bitSize := unsafe.Sizeof(value) * 8

	// This overflow is fine, tests validate the behavior when
	// encoding math.MinInt64 and math.MaxInt64.
	//gosec:disable G115
	uValue := (uint64(value) << 1) ^ uint64(value>>(bitSize-1))
	enc.Uint(uValue)
}

// Int decodes a zigzgag encoded value stored in a big-endian variable length integer
// encoded value that's read from the decoders reader. Up to 10 bytes are read from
// the decoders reader. If the decoded value overflows int64 ErrReadIntOverflow is
// returned.
func (dec *Decoder) Int() (int64, int, error) {
	uValue, n, err := dec.Uint()
	if errors.Is(err, ErrReadUintInvalid) {
		return 0, n, ErrReadIntInvalid
	} else if errors.Is(err, ErrReadUintOverflow) {
		return 0, n, ErrReadIntOverflow
	} else if err != nil {
		return 0, n, fmt.Errorf("binfmt: failed to read int: %w", errors.Unwrap(err))
	}

	// This theoretical overflow is fine, the conversion will lead
	// to sign extension bits being truncated but the calculation
	// ensures the value will be within the signed range.
	//gosec:disable G115
	value := int64((uValue >> 1) ^ (-(uValue & 1)))

	return value, n, err
}
