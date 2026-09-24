package binfmt

import (
	"errors"
	"fmt"
	"io"
)

const (
	// SetRowIdent is the row identifier for a set row.
	SetRowIdent = 'S'
)

// SetRow encodes a set row with key and value adding it to the encoder.
func (enc *Encoder) SetRow[T ValueType[V], V any](key string, value V) error {
	enc.Buffer.WriteByte(SetRowIdent)

	var vtString String

	err := vtString.Encode(enc, key)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode key: %w", err)
	}

	var t T
	enc.Buffer.WriteByte(t.Ident())

	err = t.Encode(enc, value)
	if err != nil {
		return fmt.Errorf("binfmt: failed to encode value: %w", err)
	}

	return nil
}

// SetRow decodes a set row from the decoders reader, expecting that the row
// identifier has already been decoded to determine that the row is a set row.
// The returned value byte slice is only valid until the next call made to
// the Decoder. If the set row contains an invalid value type identifier,
// ErrRowValueIdentInvalid is returned.
func (dec *Decoder) SetRow[T ValueType[V], V any]() (string, V, int, error) {
	var (
		t        T
		v        V
		vtString String
	)

	key, keyn, err := vtString.Decode(dec)
	if err != nil {
		return "", v, keyn, fmt.Errorf("binfmt: failed to decode key: %w", err)
	}

	valueIdent, err := dec.reader.ReadByte()
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}

	if err != nil {
		return "", v, keyn, fmt.Errorf("binfmt: failed to decode value type identifier: %w", err)
	}

	if valueIdent != t.Ident() {
		return "", v, keyn + 1, ErrRowValueIdentInvalid
	}

	v, n, err := t.Decode(dec)
	if err != nil {
		return "", v, keyn + 1 + n, fmt.Errorf("binfmt: failed to decode value: %w", err)
	}

	return key, v, keyn + 1 + n, nil
}
