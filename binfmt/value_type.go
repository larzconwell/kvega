package binfmt

// ValueType describes a type that can encode and
// decode values of type T. Keep in sync with type
// alias in kvega package.
type ValueType[T any] interface {
	Ident() byte
	Encode(enc *Encoder, value T) error
	Decode(dec *Decoder, createCopy bool) (T, int, error)
}
