package e2e_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestWriteAt_ThenReadAt(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	want := filledBlock(0xAA)
	if _, err := bd.WriteAt(want, blockdev.BlockSize); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	got := make([]byte, blockdev.BlockSize)
	if _, err := bd.ReadAt(got, blockdev.BlockSize); err != nil {
		t.Fatalf("ReadAt: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("read after write: first byte 0x%02x, want 0xAA", got[0])
	}
}

func TestWriteAt_Overwrite_LastWins(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xBB), blockdev.BlockSize)
	got := make([]byte, blockdev.BlockSize)
	bd.ReadAt(got, blockdev.BlockSize)
	if got[0] != 0xBB {
		t.Errorf("first byte 0x%02x, want 0xBB", got[0])
	}
}

func TestWriteAt_MultiBlock(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	p := append(filledBlock(0xCC), filledBlock(0xDD)...)
	if _, err := bd.WriteAt(p, blockdev.BlockSize); err != nil {
		t.Fatalf("WriteAt: %v", err)
	}
	got := make([]byte, 2*blockdev.BlockSize)
	bd.ReadAt(got, blockdev.BlockSize)
	for i := 0; i < blockdev.BlockSize; i++ {
		if got[i] != 0xCC {
			t.Fatalf("got[%d]=0x%02x, want 0xCC", i, got[i])
		}
	}
	for i := blockdev.BlockSize; i < 2*blockdev.BlockSize; i++ {
		if got[i] != 0xDD {
			t.Fatalf("got[%d]=0x%02x, want 0xDD", i, got[i])
		}
	}
}

func TestWriteAt_DoesNotMutateBase(t *testing.T) {
	base := patternedBase(4)
	original := append([]byte(nil), base...)
	bd, _ := blockdev.New(base)
	bd.WriteAt(filledBlock(0xFF), 0)
	bd.WriteAt(filledBlock(0xFF), blockdev.BlockSize)
	if !bytes.Equal(base, original) {
		t.Error("base was mutated after WriteAt")
	}
}

func TestWriteAt_CallerCanReuseBuffer(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, blockdev.BlockSize))
	p := filledBlock(0xAA)
	bd.WriteAt(p, 0)
	for i := range p {
		p[i] = 0xFF
	}
	got := make([]byte, blockdev.BlockSize)
	bd.ReadAt(got, 0)
	for i, b := range got {
		if b != 0xAA {
			t.Fatalf("got[%d]=0x%02x, want 0xAA (overlay was aliased)", i, b)
		}
	}
}

// Demonstrates the COW core: reads serve overlay where written, base elsewhere.
func TestReadAt_MixedBaseAndOverlay(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	bd.WriteAt(filledBlock(0xFF), 2*blockdev.BlockSize)

	p := make([]byte, 4*blockdev.BlockSize)
	bd.ReadAt(p, 0)

	for block, want := range []byte{0x00, 0x01, 0xFF, 0x03} {
		for i := 0; i < blockdev.BlockSize; i++ {
			got := p[block*blockdev.BlockSize+i]
			if got != want {
				t.Fatalf("block %d byte %d = 0x%02x, want 0x%02x", block, i, got, want)
			}
		}
	}
}

func TestWriteAt_Misaligned(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize))
	cases := []struct{ buflen, off int64 }{
		{blockdev.BlockSize, 1}, {blockdev.BlockSize, 7}, {0, 7}, {100, 0}, {4097, 0},
	}
	for _, c := range cases {
		p := make([]byte, c.buflen)
		n, err := bd.WriteAt(p, c.off)
		if !errors.Is(err, blockdev.ErrMisaligned) {
			t.Errorf("off=%d len=%d: err = %v, want ErrMisaligned", c.off, c.buflen, err)
		}
		if n != 0 {
			t.Errorf("n = %d, want 0 on error", n)
		}
	}
}

func TestWriteAt_OutOfBounds(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize))
	cases := []struct{ buflen, off int64 }{
		{blockdev.BlockSize, -1}, {blockdev.BlockSize, 4 * blockdev.BlockSize},
		{2 * blockdev.BlockSize, 3 * blockdev.BlockSize},
	}
	for _, c := range cases {
		p := make([]byte, c.buflen)
		n, err := bd.WriteAt(p, c.off)
		if !errors.Is(err, blockdev.ErrOutOfBounds) {
			t.Errorf("off=%d len=%d: err = %v, want ErrOutOfBounds", c.off, c.buflen, err)
		}
		if n != 0 {
			t.Errorf("n = %d, want 0 on error", n)
		}
	}
}
