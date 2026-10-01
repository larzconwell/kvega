package binfmt

import (
	"errors"
	"fmt"
	"io"
)

// ErrRowValueIdentInvalid is returned when a row is decoded that has an invalid value type identifier.
var ErrRowValueIdentInvalid = errors.New("binfmt: row has invalid value type identifier")

// ValueTypeMismatchError is returned when getting a value
// that has a different value type than expected.
type ValueTypeMismatchError struct {
	Expected byte
	Found    byte
}

func (vtme ValueTypeMismatchError) Error() string {
	return fmt.Sprintf(`binfmt: expected value type %q but found %q instead`, vtme.Expected, vtme.Found)
}

// RowValueDecoder is used by [Decoder.Row] to enable decoding a rows value,
// or skip it depending on what the caller wants to do once they've received
// the row identifier and key.
type RowValueDecoder struct {
	HasValue bool
	dec      *Decoder
}

// Get decodes a value from the decoders reader using the provided [ValueType] parameter.
//
// [ValueTypeMismatchError] is returned if the value type identifier s not valid for
// the provided [ValueType] and the value type identifier is then unread to enable
// the caller to inspect, retry, etc.
func (rvd RowValueDecoder) Get[VT ValueType[T], T any]() (T, int, error) {
	var (
		vt    VT
		value T
	)

	ident, err := rvd.dec.reader.ReadByte()
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return value, 0, fmt.Errorf("binfmt: failed to decode value type identifier: %w", err)
	}

	if ident != vt.Ident() {
		// This error is less important than returning the primary error.
		//nolint:errcheck
		//gosec:disable G104
		rvd.dec.reader.UnreadByte()

		return value, 1, ValueTypeMismatchError{
			Expected: vt.Ident(),
			Found:    ident,
		}
	}

	value, n, err := vt.Decode(rvd.dec, true)
	if err != nil {
		return value, 1 + n, fmt.Errorf("binfmt: failed to decode value: %w", err)
	}

	return value, 1 + n, nil
}

// Skip decodes a value from the decoders reader and discards it.
//
// [ErrRowValueIdentInvalid] is returned if the value type identifier is
// not valid, the read type value identifier is then unread to enable the
// caller to inspect, retry, etc.
func (rvd RowValueDecoder) Skip() (int, error) {
	ident, err := rvd.dec.reader.ReadByte()
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return 0, fmt.Errorf("binfmt: failed to decode value type identifier: %w", err)
	}

	var (
		vtUint   Uint
		vtInt    Int
		vtBinary Binary
		vtString String
		n        int
	)

	switch ident {
	case vtUint.Ident():
		_, n, err = vtUint.Decode(rvd.dec, false)
	case vtInt.Ident():
		_, n, err = vtInt.Decode(rvd.dec, false)
	case vtBinary.Ident():
		_, n, err = vtBinary.Decode(rvd.dec, false)
	case vtString.Ident():
		_, n, err = vtString.Decode(rvd.dec, false)
	default:
		// This error is less important than returning the primary error.
		//nolint:errcheck
		//gosec:disable G104
		rvd.dec.reader.UnreadByte()

		return 1, ErrRowValueIdentInvalid
	}

	if err != nil {
		return 1 + n, fmt.Errorf("binfmt: failed to decode value: %w", err)
	}

	return 1 + n, nil
}
