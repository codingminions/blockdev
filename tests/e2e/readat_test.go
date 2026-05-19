package e2e_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestReadAt_FromBase(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	for block := 0; block < 4; block++ {
		p := make([]byte, blockdev.BlockSize)
		n, err := bd.ReadAt(p, int64(block)*blockdev.BlockSize)
		if err != nil {
			t.Fatalf("block %d: ReadAt: %v", block, err)
		}
		if n != blockdev.BlockSize {
			t.Errorf("block %d: n = %d, want %d", block, n, blockdev.BlockSize)
		}
		want := bytes.Repeat([]byte{byte(block)}, blockdev.BlockSize)
		if !bytes.Equal(p, want) {
			t.Errorf("block %d: first byte = 0x%02x, want 0x%02x", block, p[0], byte(block))
		}
	}
}

func TestReadAt_MultiBlock(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	p := make([]byte, 2*blockdev.BlockSize)
	if _, err := bd.ReadAt(p, blockdev.BlockSize); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < blockdev.BlockSize; i++ {
		if p[i] != 1 {
			t.Fatalf("p[%d]=0x%02x, want 0x01", i, p[i])
		}
	}
	for i := blockdev.BlockSize; i < 2*blockdev.BlockSize; i++ {
		if p[i] != 2 {
			t.Fatalf("p[%d]=0x%02x, want 0x02", i, p[i])
		}
	}
}

func TestReadAt_Empty(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	for _, off := range []int64{0, blockdev.BlockSize, 4 * blockdev.BlockSize} {
		n, err := bd.ReadAt(nil, off)
		if err != nil {
			t.Errorf("off=%d: %v", off, err)
		}
		if n != 0 {
			t.Errorf("off=%d: n = %d, want 0", off, n)
		}
	}
}

func TestReadAt_Misaligned(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	cases := []struct{ buflen, off int64 }{
		{blockdev.BlockSize, 1}, {blockdev.BlockSize, 7}, {blockdev.BlockSize, 4097},
		{0, 7}, {100, 0}, {4097, 0},
	}
	for _, c := range cases {
		n, err := bd.ReadAt(make([]byte, c.buflen), c.off)
		if !errors.Is(err, blockdev.ErrMisaligned) {
			t.Errorf("off=%d len=%d: err = %v, want ErrMisaligned", c.off, c.buflen, err)
		}
		if n != 0 {
			t.Errorf("off=%d len=%d: n = %d, want 0 on error", c.off, c.buflen, n)
		}
	}
}

func TestReadAt_OutOfBounds(t *testing.T) {
	bd, _ := blockdev.New(patternedBase(4))
	cases := []struct{ buflen, off int64 }{
		{blockdev.BlockSize, -1},
		{blockdev.BlockSize, 4 * blockdev.BlockSize},
		{2 * blockdev.BlockSize, 3 * blockdev.BlockSize},
		{blockdev.BlockSize, 100 * blockdev.BlockSize},
	}
	for _, c := range cases {
		n, err := bd.ReadAt(make([]byte, c.buflen), c.off)
		if !errors.Is(err, blockdev.ErrOutOfBounds) {
			t.Errorf("off=%d len=%d: err = %v, want ErrOutOfBounds", c.off, c.buflen, err)
		}
		if n != 0 {
			t.Errorf("off=%d len=%d: n = %d, want 0 on error", c.off, c.buflen, n)
		}
	}
}
