package e2e_test

import (
	"math/rand"
	"testing"

	"github.com/codingminions/blockdev"
)

// Random writes, then Serialize → Deserialize → assert ReadAt parity.
// Seeded RNG so failures reproduce.
func TestRoundTrip_RandomWrites(t *testing.T) {
	const blocks = 64
	const ops = 200
	rng := rand.New(rand.NewSource(42))

	base := make([]byte, blocks*blockdev.BlockSize)
	rng.Read(base)

	bd, _ := blockdev.New(base)
	for i := 0; i < ops; i++ {
		block := rng.Int63n(blocks)
		data := make([]byte, blockdev.BlockSize)
		rng.Read(data)
		bd.WriteAt(data, block*blockdev.BlockSize)
	}

	blob := bd.Serialize()
	bd2, err := blockdev.Deserialize(blob, base)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	for block := int64(0); block < blocks; block++ {
		var a, b [4096]byte
		bd.ReadAt(a[:], block*blockdev.BlockSize)
		bd2.ReadAt(b[:], block*blockdev.BlockSize)
		if a != b {
			t.Fatalf("round-trip mismatch at block %d", block)
		}
	}
}
