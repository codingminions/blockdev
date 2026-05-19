package e2e_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestSerialize_EmptyOverlay(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize))
	if blob := bd.Serialize(); blob != nil {
		t.Errorf("Serialize() = %d bytes, want nil for empty overlay", len(blob))
	}
}

func TestSerialize_SingleBlock(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize))
	bd.WriteAt(filledBlock(0xAA), 2*blockdev.BlockSize)

	blob := bd.Serialize()
	if len(blob) != blockdev.EntrySize {
		t.Fatalf("len(blob) = %d, want %d", len(blob), blockdev.EntrySize)
	}
	if block := binary.BigEndian.Uint64(blob[:8]); block != 2 {
		t.Errorf("encoded block# = %d, want 2", block)
	}
	for i := 8; i < blockdev.EntrySize; i++ {
		if blob[i] != 0xAA {
			t.Fatalf("blob[%d] = 0x%02x, want 0xAA", i, blob[i])
		}
	}
}

// Encoder must sort even when writes arrived out of order.
func TestSerialize_MultipleBlocks_Sorted(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 8*blockdev.BlockSize))
	bd.WriteAt(filledBlock(0x05), 5*blockdev.BlockSize)
	bd.WriteAt(filledBlock(0x01), 1*blockdev.BlockSize)
	bd.WriteAt(filledBlock(0x07), 7*blockdev.BlockSize)
	bd.WriteAt(filledBlock(0x03), 3*blockdev.BlockSize)

	blob := bd.Serialize()
	if len(blob) != 4*blockdev.EntrySize {
		t.Fatalf("len(blob) = %d, want %d", len(blob), 4*blockdev.EntrySize)
	}
	wantOrder := []int64{1, 3, 5, 7}
	for i, want := range wantOrder {
		got := int64(binary.BigEndian.Uint64(blob[i*blockdev.EntrySize:]))
		if got != want {
			t.Errorf("entry %d block# = %d, want %d", i, got, want)
		}
	}
}

// Same overlay state must produce identical bytes regardless of write order.
func TestSerialize_Deterministic(t *testing.T) {
	build := func() []byte {
		bd, _ := blockdev.New(make([]byte, 8*blockdev.BlockSize))
		bd.WriteAt(filledBlock(0x05), 5*blockdev.BlockSize)
		bd.WriteAt(filledBlock(0x01), 1*blockdev.BlockSize)
		bd.WriteAt(filledBlock(0x03), 3*blockdev.BlockSize)
		return bd.Serialize()
	}
	if !bytes.Equal(build(), build()) {
		t.Errorf("Serialize is non-deterministic across builds")
	}

	bd1, _ := blockdev.New(make([]byte, 8*blockdev.BlockSize))
	bd1.WriteAt(filledBlock(0xAA), 0)
	bd1.WriteAt(filledBlock(0xBB), 4*blockdev.BlockSize)

	bd2, _ := blockdev.New(make([]byte, 8*blockdev.BlockSize))
	bd2.WriteAt(filledBlock(0xBB), 4*blockdev.BlockSize)
	bd2.WriteAt(filledBlock(0xAA), 0)

	if !bytes.Equal(bd1.Serialize(), bd2.Serialize()) {
		t.Errorf("write order affected Serialize output")
	}
}

// Locks the wire format byte-for-byte; any format change must update this test.
func TestSerialize_Golden(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize))
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xBB), 3*blockdev.BlockSize)

	got := bd.Serialize()

	want := make([]byte, 2*blockdev.EntrySize)
	binary.BigEndian.PutUint64(want[0:8], 1)
	copy(want[8:8+blockdev.BlockSize], filledBlock(0xAA))
	binary.BigEndian.PutUint64(want[blockdev.EntrySize:blockdev.EntrySize+8], 3)
	copy(want[blockdev.EntrySize+8:], filledBlock(0xBB))

	if !bytes.Equal(got, want) {
		t.Errorf("Serialize output does not match golden bytes")
	}
}

func TestSerialize_Observer(t *testing.T) {
	var got blockdev.Event
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) {
			if e.Op == blockdev.OpSerialize {
				got = e
			}
		}),
	)
	bd.WriteAt(filledBlock(0xAA), 0)
	bd.WriteAt(filledBlock(0xBB), blockdev.BlockSize)

	blob := bd.Serialize()

	if got.Op != blockdev.OpSerialize {
		t.Errorf("op = %v, want OpSerialize", got.Op)
	}
	if got.Length != len(blob) {
		t.Errorf("Length = %d, want %d", got.Length, len(blob))
	}
	if got.Blocks != 2 {
		t.Errorf("Blocks = %d, want 2", got.Blocks)
	}
}
