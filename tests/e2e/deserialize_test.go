package e2e_test

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

// Both nil and []byte{} must produce a base-only device.
func TestDeserialize_Empty(t *testing.T) {
	base := patternedBase(4)
	for _, c := range []struct {
		name string
		blob []byte
	}{
		{"nil", nil},
		{"empty slice", []byte{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			bd, err := blockdev.Deserialize(c.blob, base)
			if err != nil {
				t.Fatalf("Deserialize: %v", err)
			}
			p := make([]byte, blockdev.BlockSize)
			bd.ReadAt(p, blockdev.BlockSize)
			for i, b := range p {
				if b != 1 {
					t.Fatalf("p[%d]=0x%02x, want 0x01", i, b)
				}
			}
		})
	}
}

// Write → Serialize → Deserialize → reads must match the original device.
func TestDeserialize_RoundTrip(t *testing.T) {
	base := patternedBase(8)
	bd, _ := blockdev.New(base)
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xBB), 4*blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xCC), 6*blockdev.BlockSize)

	blob := bd.Serialize()
	bd2, err := blockdev.Deserialize(blob, base)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	for block := int64(0); block < 8; block++ {
		var a, b [4096]byte
		bd.ReadAt(a[:], block*blockdev.BlockSize)
		bd2.ReadAt(b[:], block*blockdev.BlockSize)
		if a != b {
			t.Fatalf("round-trip mismatch at block %d", block)
		}
	}
}

// Every malformed-blob shape our encoder wouldn't produce must yield ErrBadFormat.
func TestDeserialize_Rejects(t *testing.T) {
	base := patternedBase(4)

	badLength := make([]byte, 7)

	outOfRange := make([]byte, blockdev.EntrySize)
	binary.BigEndian.PutUint64(outOfRange[0:8], 99)

	negativeBlock := make([]byte, blockdev.EntrySize)
	binary.BigEndian.PutUint64(negativeBlock[0:8], 1<<63) // high bit → negative as int64

	duplicate := make([]byte, 2*blockdev.EntrySize)
	binary.BigEndian.PutUint64(duplicate[0:8], 1)
	binary.BigEndian.PutUint64(duplicate[blockdev.EntrySize:blockdev.EntrySize+8], 1)

	unsorted := make([]byte, 2*blockdev.EntrySize)
	binary.BigEndian.PutUint64(unsorted[0:8], 2)
	binary.BigEndian.PutUint64(unsorted[blockdev.EntrySize:blockdev.EntrySize+8], 1)

	for _, c := range []struct {
		name string
		blob []byte
	}{
		{"bad length", badLength},
		{"block# out of range", outOfRange},
		{"block# negative", negativeBlock},
		{"duplicate block#", duplicate},
		{"unsorted entries", unsorted},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := blockdev.Deserialize(c.blob, base); !errors.Is(err, blockdev.ErrBadFormat) {
				t.Errorf("err = %v, want ErrBadFormat", err)
			}
		})
	}
}

// Misaligned initial is a separate failure mode (ErrMisaligned, not ErrBadFormat).
func TestDeserialize_RejectsMisalignedInitial(t *testing.T) {
	if _, err := blockdev.Deserialize(nil, make([]byte, 100)); !errors.Is(err, blockdev.ErrMisaligned) {
		t.Errorf("err = %v, want ErrMisaligned", err)
	}
}

func TestDeserialize_AppliesOptions(t *testing.T) {
	var got blockdev.Event
	_, err := blockdev.Deserialize(nil, make([]byte, blockdev.BlockSize),
		blockdev.WithName("snap"),
		blockdev.WithObserver(func(e blockdev.Event) {
			if e.Op == blockdev.OpDeserialize {
				got = e
			}
		}),
	)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if got.Op != blockdev.OpDeserialize || got.Device != "snap" {
		t.Errorf("observer event = %+v", got)
	}
}

// Observer must still fire when Deserialize fails, with Err populated.
func TestDeserialize_Observer_FiresOnError(t *testing.T) {
	var got blockdev.Event
	blob := make([]byte, 7)
	_, err := blockdev.Deserialize(blob, make([]byte, blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) {
			if e.Op == blockdev.OpDeserialize {
				got = e
			}
		}),
	)
	if !errors.Is(err, blockdev.ErrBadFormat) {
		t.Fatalf("err = %v, want ErrBadFormat", err)
	}
	if got.Op != blockdev.OpDeserialize {
		t.Errorf("op = %v, want OpDeserialize", got.Op)
	}
	if !errors.Is(got.Err, blockdev.ErrBadFormat) {
		t.Errorf("event err = %v, want ErrBadFormat", got.Err)
	}
}
