package binfmt

import (
	"errors"
	"fmt"
	"io"
)

// ErrRowValueIdentInvalid is returned when a row is decoded that has an invalid value type identifier.
var ErrRowValueIdentInvalid = errors.New("binfmt: row has invalid value type identifier")

// RowValueDecoder is used by [Decoder.Row] to enable decoding a rows value,
// or skip it depending on what the caller wants to do once they've received
// the row identifier and key.
type RowValueDecoder struct {
	HasValue bool
	dec      *Decoder
}

// Get decodes a value from the decoders reader using the provided
// [ValueType] parameter. [ErrRowValueIdentInvalid] is returned if
// the value type identifier is not valid for the provided [ValueType].
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
		return value, 1, ErrRowValueIdentInvalid
	}

	value, n, err := vt.Decode(rvd.dec)
	if err != nil {
		return value, 1 + n, fmt.Errorf("binfmt: failed to decode value: %w", err)
	}

	return value, 1 + n, nil
}
