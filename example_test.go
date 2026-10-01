package kvega_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/larzconwell/kvega"
)

func ExampleEmbeddedDB() {
	path := filepath.Join(os.TempDir(), "example.kvega")

	db, err := kvega.OpenEmbeddedDB(path)
	if err != nil {
		fmt.Println("failed to open db:", err)
		return
	}

	defer func() {
		err := db.Close()
		if err != nil {
			fmt.Println("failed to close db:", err)
		}
	}()

	err = db.Set[kvega.String]("key", "value")
	if err != nil {
		fmt.Println("failed to set key:", err)
		return
	}

	err = db.Delete("key")
	if err != nil {
		fmt.Println("failed to delete key:", err)
		return
	}

	err = db.Set[kvega.Uint]("key", 5)
	if err != nil {
		fmt.Println("failed to set key:", err)
		return
	}

	val, err := db.Get[kvega.Uint]("key")
	if err != nil {
		fmt.Println("failed to get key:", err)
		return
	}

	fmt.Println("value:", val)

	// Output: value: 5
}
