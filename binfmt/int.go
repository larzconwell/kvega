package binfmt

import (
	"errors"
	"fmt"
	"unsafe"
)

var (
	// ErrDecodeIntInvalid is returned when an int cannot be decoded due to invalid reader data.
	ErrDecodeIntInvalid = errors.New("binfmt: decode int failed due to invalid reader data")
	// ErrDecodeIntOverflow is returned when a decoded int is larger than int64.
	ErrDecodeIntOverflow = errors.New("binfmt: decoded int does not fit in int64")
)

var _ ValueType[int64] = Int{}

// Int implements ValueType[int64] and is able
// to encode and decode int64 values.
type Int struct{}

// Ident returns the Int identifier.
func (Int) Ident() byte {
	return 'I'
}

// Encode encodes value using zigzag encoding along with big-endian variable length integer
// encoding and adds it to the encoder. 1 to 10 bytes may be added to the encoder
// depending on the value.
//
// The first step is zigzag encoding which is a method to represent a signed integer
// as an unsigned integer by evenly spreading positive and negative values across the
// unsigned integer range. Negative values become double their absolute value minus
// one. Positive values meanwhile become double their absolute value.
//
// After the int64 has been converted to a uint64, big-endian variable length
// integer encoding is done, context to which can be gathered by reading Uint.Encode.
func (Int) Encode(enc *Encoder, value int64) error {
	bitSize := unsafe.Sizeof(value) * 8

	// This overflow is fine, tests validate the behavior when
	// encoding math.MinInt64 and math.MaxInt64.
	//gosec:disable G115
	uValue := (uint64(value) << 1) ^ uint64(value>>(bitSize-1))

	var vtUint = Uint{}

	// Encoding uint does not return error.
	//nolint:errcheck
	//gosec:disable G104
	vtUint.Encode(enc, uValue)

	return nil
}

// Decode decodes a zigzgag encoded value stored in a big-endian variable length integer
// encoded value that's read from the decoders reader. Up to 10 bytes are read from
// the decoders reader. If the decoded value overflows int64 ErrDecodeIntOverflow is
// returned.
func (Int) Decode(dec *Decoder) (int64, int, error) {
	var vtUint = Uint{}

	uValue, n, err := vtUint.Decode(dec)
	if errors.Is(err, ErrDecodeUintInvalid) {
		return 0, n, ErrDecodeIntInvalid
	} else if errors.Is(err, ErrDecodeUintOverflow) {
		return 0, n, ErrDecodeIntOverflow
	} else if err != nil {
		return 0, n, fmt.Errorf("binfmt: failed to decode int: %w", errors.Unwrap(err))
	}

	// This theoretical overflow is fine, the conversion will lead
	// to sign extension bits being truncated but the calculation
	// ensures the value will be within the signed range.
	//gosec:disable G115
	value := int64((uValue >> 1) ^ (-(uValue & 1)))

	return value, n, nil
}
