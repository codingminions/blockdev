package e2e_test

import (
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

// Covers the firing contract: post-op event with correct fields, on success
// and on failure, with Err set when the op failed.
func TestObserver_Fires(t *testing.T) {
	var events []blockdev.Event
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize),
		blockdev.WithName("agent-1"),
		blockdev.WithObserver(func(e blockdev.Event) { events = append(events, e) }),
	)

	// Success: read + write.
	bd.ReadAt(make([]byte, blockdev.BlockSize), 0)
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)

	// Failure: misaligned (in-bounds) read fires too, with Err.
	bd.ReadAt(make([]byte, blockdev.BlockSize), 7)

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].Op != blockdev.OpRead || events[0].Device != "agent-1" || events[0].Err != nil {
		t.Errorf("read event = %+v", events[0])
	}
	if events[1].Op != blockdev.OpWrite || events[1].Blocks != 1 || events[1].Err != nil {
		t.Errorf("write event = %+v", events[1])
	}
	if events[2].Op != blockdev.OpRead || !errors.Is(events[2].Err, blockdev.ErrMisaligned) {
		t.Errorf("misaligned event = %+v", events[2])
	}
	if events[0].Duration <= 0 {
		t.Errorf("duration not measured: %v", events[0].Duration)
	}
}

// A user observer that panics must not crash the I/O path.
func TestObserver_RecoversFromPanic(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) { panic("observer is broken") }),
	)
	n, err := bd.ReadAt(make([]byte, blockdev.BlockSize), 0)
	if err != nil {
		t.Errorf("err = %v, want nil despite observer panic", err)
	}
	if n != blockdev.BlockSize {
		t.Errorf("n = %d, want %d", n, blockdev.BlockSize)
	}
}

// Zero-observer is the zero-overhead path; must work without firing anything.
func TestObserver_NotInvokedWhenAbsent(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, blockdev.BlockSize))
	if _, err := bd.ReadAt(make([]byte, blockdev.BlockSize), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := bd.WriteAt(make([]byte, blockdev.BlockSize), 0); err != nil {
		t.Fatal(err)
	}
}
