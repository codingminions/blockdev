package e2e_test

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/codingminions/blockdev"
)

// Both nil and []byte{} produce a base-only device.
func TestDeserialize_Empty(t *testing.T) {
	base := patternedBase(4)
	for _, blob := range [][]byte{nil, {}} {
		bd, err := blockdev.Deserialize(blob, base)
		if err != nil {
			t.Fatalf("Deserialize(%v): %v", blob, err)
		}
		p := make([]byte, blockdev.BlockSize)
		bd.ReadAt(p, blockdev.BlockSize)
		for i, b := range p {
			if b != 1 {
				t.Fatalf("blob=%v p[%d]=0x%02x, want 0x01", blob, i, b)
			}
		}
	}
}

// Write → Serialize → Deserialize → reads must match original.
func TestDeserialize_RoundTrip(t *testing.T) {
	base := patternedBase(8)
	bd, _ := blockdev.New(base)
	bd.WriteAt(filledBlock(0xAA), blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xBB), 4*blockdev.BlockSize)
	bd.WriteAt(filledBlock(0xCC), 6*blockdev.BlockSize)

	bd2, err := blockdev.Deserialize(bd.Serialize(), base)
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

// Every malformed-blob shape our encoder wouldn't produce → ErrBadFormat.
func TestDeserialize_Rejects(t *testing.T) {
	base := patternedBase(4)

	badLength := make([]byte, 7)

	outOfRange := make([]byte, blockdev.EntrySize)
	binary.BigEndian.PutUint64(outOfRange[0:8], 99)

	negativeBlock := make([]byte, blockdev.EntrySize)
	binary.BigEndian.PutUint64(negativeBlock[0:8], 1<<63)

	duplicate := make([]byte, 2*blockdev.EntrySize)
	binary.BigEndian.PutUint64(duplicate[0:8], 1)
	binary.BigEndian.PutUint64(duplicate[blockdev.EntrySize:blockdev.EntrySize+8], 1)

	unsorted := make([]byte, 2*blockdev.EntrySize)
	binary.BigEndian.PutUint64(unsorted[0:8], 2)
	binary.BigEndian.PutUint64(unsorted[blockdev.EntrySize:blockdev.EntrySize+8], 1)

	cases := []struct {
		name string
		blob []byte
	}{
		{"bad length", badLength},
		{"out of range", outOfRange},
		{"negative block#", negativeBlock},
		{"duplicate", duplicate},
		{"unsorted", unsorted},
	}
	for _, c := range cases {
		if _, err := blockdev.Deserialize(c.blob, base); !errors.Is(err, blockdev.ErrBadFormat) {
			t.Errorf("%s: err = %v, want ErrBadFormat", c.name, err)
		}
	}
}

// Misaligned initial is a different failure mode (ErrMisaligned, not ErrBadFormat).
func TestDeserialize_RejectsMisalignedInitial(t *testing.T) {
	if _, err := blockdev.Deserialize(nil, make([]byte, 100)); !errors.Is(err, blockdev.ErrMisaligned) {
		t.Errorf("err = %v, want ErrMisaligned", err)
	}
}

// Options propagate through Deserialize, including observer firing on error.
func TestDeserialize_Options(t *testing.T) {
	// Happy path: name reaches observer.
	var ok blockdev.Event
	_, err := blockdev.Deserialize(nil, make([]byte, blockdev.BlockSize),
		blockdev.WithName("snap"),
		blockdev.WithObserver(func(e blockdev.Event) {
			if e.Op == blockdev.OpDeserialize {
				ok = e
			}
		}),
	)
	if err != nil {
		t.Fatalf("Deserialize (happy): %v", err)
	}
	if ok.Op != blockdev.OpDeserialize || ok.Device != "snap" {
		t.Errorf("happy event = %+v", ok)
	}

	// Failure path: observer still fires with Err populated.
	var bad blockdev.Event
	_, err = blockdev.Deserialize(make([]byte, 7), make([]byte, blockdev.BlockSize),
		blockdev.WithObserver(func(e blockdev.Event) {
			if e.Op == blockdev.OpDeserialize {
				bad = e
			}
		}),
	)
	if !errors.Is(err, blockdev.ErrBadFormat) {
		t.Fatalf("Deserialize (bad): err = %v, want ErrBadFormat", err)
	}
	if !errors.Is(bad.Err, blockdev.ErrBadFormat) {
		t.Errorf("bad event err = %v, want ErrBadFormat", bad.Err)
	}
}
