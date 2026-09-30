package binfmt_test

import (
	"bytes"
	"fmt"

	"github.com/larzconwell/kvega/binfmt"
)

func ExampleEncoder() {
	var buf bytes.Buffer

	enc := binfmt.NewEncoder()

	err := enc.SetRow[binfmt.String]("greeting", "你好")
	if err != nil {
		fmt.Println("failed to encode set row:", err)
		return
	}

	_, err = enc.WriteTo(&buf)
	if err != nil {
		fmt.Println("failed to write set row:", err)
		return
	}

	fmt.Println("row identifier:", string(buf.Bytes()[0]))
	fmt.Println("key length:", buf.Bytes()[1])
	fmt.Println("key:", string(buf.Bytes()[2:10]))
	fmt.Println("value type identifier:", string(buf.Bytes()[10]))
	fmt.Println("value length:", buf.Bytes()[11])
	fmt.Println("value:", string(buf.Bytes()[12:]))

	// Output:
	// row identifier: S
	// key length: 16
	// key: greeting
	// value type identifier: S
	// value length: 12
	// value: 你好
}

func ExampleDecoder() {
	var buf bytes.Buffer
	buf.WriteByte(binfmt.SetRowIdent)
	buf.WriteByte(16)
	buf.WriteString("greeting")
	buf.WriteByte('S')
	buf.WriteByte(12)
	buf.WriteString("你好")

	dec := binfmt.NewDecoder(&buf)

	ident, key, valueDecoder, _, err := dec.Row()
	if err != nil {
		fmt.Println("failed to decode set row:", err)
		return
	}

	value, _, err := valueDecoder.Get[binfmt.String]()
	if err != nil {
		fmt.Println("failed to decode value:", err)
		return
	}

	fmt.Println("row identifier:", string(ident))
	fmt.Println("key:", key)
	fmt.Println("value:", value)

	// Output:
	// row identifier: S
	// key: greeting
	// value: 你好
}
