package e2e_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestConcurrent_ReadsAndWrites(t *testing.T) {
	var observed atomic.Int64
	bd, _ := blockdev.New(patternedBase(16),
		blockdev.WithObserver(func(e blockdev.Event) { observed.Add(1) }),
	)

	var wg sync.WaitGroup
	const goroutines = 8
	const ops = 200

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			buf := make([]byte, blockdev.BlockSize)
			for i := 0; i < ops; i++ {
				off := int64((id*ops+i)%16) * blockdev.BlockSize
				_, _ = bd.ReadAt(buf, off)
			}
		}(g)
	}
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := filledBlock(byte(id))
			for i := 0; i < ops; i++ {
				off := int64((id*ops+i)%16) * blockdev.BlockSize
				_, _ = bd.WriteAt(data, off)
			}
		}(g)
	}
	wg.Wait()

	if observed.Load() != int64(2*goroutines*ops) {
		t.Errorf("observer fired %d times, want %d", observed.Load(), 2*goroutines*ops)
	}
}
