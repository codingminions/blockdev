package e2e_test

import (
	"errors"
	"testing"
	"time"

	"github.com/codingminions/blockdev"
)

func TestObserver_FiresOnReadAndWrite(t *testing.T) {
	var events []blockdev.Event
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize),
		blockdev.WithName("agent-1"),
		blockdev.WithObserver(func(e blockdev.Event) { events = append(events, e) }),
	)
	bd.ReadAt(make([]byte, blockdev.BlockSize), 0)
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Op != blockdev.OpRead || events[0].Length != blockdev.BlockSize ||
		events[0].Blocks != 1 || events[0].Device != "agent-1" {
		t.Errorf("read event = %+v", events[0])
	}
	if events[1].Op != blockdev.OpWrite || events[1].Length != blockdev.BlockSize ||
		events[1].Offset != blockdev.BlockSize {
		t.Errorf("write event = %+v", events[1])
	}
	if events[0].Err != nil || events[1].Err != nil {
		t.Errorf("events report error on success: %+v, %+v", events[0].Err, events[1].Err)
	}
}

func TestObserver_FiresOnError(t *testing.T) {
	var got blockdev.Event
	bd, _ := blockdev.New(make([]byte, 4*blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) { got = e }),
	)
	bd.ReadAt(make([]byte, blockdev.BlockSize), 7) // misaligned, in bounds

	if got.Op != blockdev.OpRead {
		t.Errorf("op = %v, want OpRead", got.Op)
	}
	if !errors.Is(got.Err, blockdev.ErrMisaligned) {
		t.Errorf("err = %v, want ErrMisaligned", got.Err)
	}
}

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

func TestObserver_DurationIsMeasured(t *testing.T) {
	var got blockdev.Event
	bd, _ := blockdev.New(make([]byte, blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) { got = e }),
	)
	bd.ReadAt(make([]byte, blockdev.BlockSize), 0)
	if got.Duration <= 0 {
		t.Errorf("duration = %v, want > 0", got.Duration)
	}
	if got.Duration > time.Second {
		t.Errorf("duration = %v, suspiciously long", got.Duration)
	}
}

func TestObserver_NotInvokedWhenAbsent(t *testing.T) {
	bd, _ := blockdev.New(make([]byte, blockdev.BlockSize))
	if _, err := bd.ReadAt(make([]byte, blockdev.BlockSize), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := bd.WriteAt(make([]byte, blockdev.BlockSize), 0); err != nil {
		t.Fatal(err)
	}
}
