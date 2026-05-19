package e2e_test

import (
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

func TestNew_AcceptsAlignedBase(t *testing.T) {
	for _, size := range []int{0, blockdev.BlockSize, 16 * blockdev.BlockSize} {
		if _, err := blockdev.New(make([]byte, size)); err != nil {
			t.Errorf("New(%d): %v", size, err)
		}
	}
}

func TestNew_RejectsMisalignedBase(t *testing.T) {
	for _, size := range []int{1, 4095, 4097} {
		_, err := blockdev.New(make([]byte, size))
		if !errors.Is(err, blockdev.ErrMisaligned) {
			t.Errorf("New(%d): err = %v, want ErrMisaligned", size, err)
		}
	}
}

// Verifying via the observer rather than poking unexported fields proves both
// that WithName stuck and that WithObserver fires correctly.
func TestNew_AppliesOptions(t *testing.T) {
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
