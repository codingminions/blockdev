package e2e_test

import (
	"bytes"

	"github.com/codingminions/blockdev"
)

// Each byte equals its block number, so reads are self-identifying.
func patternedBase(blocks int) []byte {
	b := make([]byte, blocks*blockdev.BlockSize)
	for i := range b {
		b[i] = byte(i / blockdev.BlockSize)
	}
	return b
}

func filledBlock(value byte) []byte {
	return bytes.Repeat([]byte{value}, blockdev.BlockSize)
}
