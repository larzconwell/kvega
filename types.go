package kvega

import (
	"github.com/larzconwell/kvega/binfmt"
)

type (
	// ValueType describes a type that can encode and
	// decode values of type T.
	ValueType[T any] = binfmt.ValueType[T]

	// Uint implements [ValueType][uint64] and is able to
	// encode and decode uint64 values.
	Uint = binfmt.Uint

	// Int implements [ValueType][int64] and is able
	// to encode and decode int64 values.
	Int = binfmt.Int

	// Binary implements [ValueType][[]byte] and is able to
	// encode and decode []byte values.
	Binary = binfmt.Binary

	// String implements [ValueType][string] and is able to
	// encode and decode string values.
	String = binfmt.String
)
