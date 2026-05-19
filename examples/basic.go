//go:build ignore

// basic.go — minimal write/read demo.
// Run directly:
//
//	go run examples/basic.go
package main

import (
	"fmt"
	"log"

	"github.com/codingminions/blockdev"
)

func main() {
	base := make([]byte, 16*blockdev.BlockSize)
	bd, err := blockdev.New(base)
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, blockdev.BlockSize)
	for i := range data {
		data[i] = 'A'
	}
	if _, err := bd.WriteAt(data, blockdev.BlockSize); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, blockdev.BlockSize)
	if _, err := bd.ReadAt(buf, blockdev.BlockSize); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Wrote 4 KB of 'A' at offset %d\n", blockdev.BlockSize)
	fmt.Printf("First byte read back: %c\n", buf[0])
	fmt.Printf("Snapshot size: %d bytes (base is %d bytes)\n",
		len(bd.Serialize()), len(base))
}
