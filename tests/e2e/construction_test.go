package e2e_test

import (
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestNew_AcceptsAlignedBase(t *testing.T) {
	for _, size := range []int{0, blockdev.BlockSize, 2 * blockdev.BlockSize, 16 * blockdev.BlockSize} {
		size := size
		t.Run("", func(t *testing.T) {
			if _, err := blockdev.New(make([]byte, size)); err != nil {
				t.Fatalf("New(%d): %v", size, err)
			}
		})
	}
}

func TestNew_RejectsMisalignedBase(t *testing.T) {
	for _, size := range []int{1, 100, 4095, 4097, blockdev.BlockSize + 1, 3*blockdev.BlockSize - 1} {
		size := size
		t.Run("", func(t *testing.T) {
			_, err := blockdev.New(make([]byte, size))
			if !errors.Is(err, blockdev.ErrMisaligned) {
				t.Errorf("New(%d): err = %v, want ErrMisaligned in chain", size, err)
			}
		})
	}
}

func TestNew_AppliesOptions(t *testing.T) {
	// Verifying via the observer rather than poking unexported fields: it
	// proves both that WithName stuck and WithObserver fires correctly.
	var got blockdev.Event
	bd, err := blockdev.New(make([]byte, blockdev.BlockSize),
		blockdev.WithName("test-device"),
		blockdev.WithObserver(func(e blockdev.Event) { got = e }),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bd.ReadAt(make([]byte, blockdev.BlockSize), 0)
	if got.Device != "test-device" {
		t.Errorf("Event.Device = %q, want test-device", got.Device)
	}
	if got.Op != blockdev.OpRead {
		t.Errorf("Event.Op = %v, want OpRead", got.Op)
	}
}
