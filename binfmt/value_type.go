package binfmt

// ValueType describes a type that can encode and
// decode values of type T.
type ValueType[T any] interface {
	Ident() byte
	Encode(enc *Encoder, value T) error
	Decode(dec *Decoder) (T, int, error)
}
