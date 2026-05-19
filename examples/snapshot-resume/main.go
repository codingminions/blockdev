package main

import (
	"fmt"
	"log"

	"github.com/codingminions/blockdev"
)

func main() {
	// Imagine this is a shared filesystem image used by many sandboxes.
	base := make([]byte, 64*blockdev.BlockSize)

	// Sandbox starts and makes some changes.
	bd, err := blockdev.New(base, blockdev.WithName("sandbox-1"))
	if err != nil {
		log.Fatal(err)
	}

	for _, block := range []int64{2, 7, 13, 29} {
		data := make([]byte, blockdev.BlockSize)
		for i := range data {
			data[i] = byte(block)
		}
		if _, err := bd.WriteAt(data, block*blockdev.BlockSize); err != nil {
			log.Fatal(err)
		}
	}

	// Pause — capture only the diff.
	blob := bd.Serialize()
	fmt.Printf("Pause: %d-byte snapshot for 4 changed blocks (base is %d bytes — would have stored %.0f%% more without COW)\n",
		len(blob), len(base), 100*float64(len(base)-len(blob))/float64(len(blob)))

	// Days later, somewhere else, resume.
	bd2, err := blockdev.Deserialize(blob, base)
	if err != nil {
		log.Fatal(err)
	}

	// Verify reads match what the original sandbox wrote.
	for _, block := range []int64{2, 7, 13, 29} {
		buf := make([]byte, blockdev.BlockSize)
		if _, err := bd2.ReadAt(buf, block*blockdev.BlockSize); err != nil {
			log.Fatal(err)
		}
		if buf[0] != byte(block) {
			log.Fatalf("mismatch at block %d", block)
		}
	}
	fmt.Println("Resume: every block matches the original sandbox.")
}
