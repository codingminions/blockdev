package blockdev

import (
	"encoding/binary"
	"fmt"
	"sort"
	"time"
)

// EntrySize is the on-wire size of one changed-block record in the serialized
// blob: 8 bytes big-endian block number + BlockSize bytes of data.
const EntrySize = 8 + BlockSize

// Serialize emits the changed blocks in ascending block-number order with no
// header or trailer. Returns nil for an empty overlay so a fresh device costs
// zero allocations.
func (b *BlockDevice) Serialize() []byte {
	var start time.Time
	if b.cfg.observer != nil {
		start = time.Now()
	}
	blob := b.serialize()
	if b.cfg.observer != nil {
		b.fireObserver(Event{
			Op:       OpSerialize,
			Device:   b.cfg.name,
			Length:   len(blob),
			Blocks:   len(blob) / EntrySize,
			Duration: time.Since(start),
		})
	}
	return blob
}

type serializedEntry struct {
	block int64
	data  []byte
}

func (b *BlockDevice) serialize() []byte {
	// Take a point-in-time snapshot under the read lock; stored slices are
	// never mutated (overlay.put replaces, never edits in place) so it's safe
	// to encode them after the lock is released.
	var entries []serializedEntry
	b.ov.each(func(block int64, data []byte) bool {
		entries = append(entries, serializedEntry{block: block, data: data})
		return true
	})
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].block < entries[j].block
	})
	blob := make([]byte, len(entries)*EntrySize)
	for i, e := range entries {
		off := i * EntrySize
		binary.BigEndian.PutUint64(blob[off:], uint64(e.block))
		copy(blob[off+8:], e.data)
	}
	return blob
}

// Deserialize reconstructs a BlockDevice from a blob produced by Serialize
// and the original initial base. Rejects any blob that doesn't match what our
// encoder produces — length not a multiple of EntrySize, block numbers out of
// range, duplicate blocks, or non-ascending order.
func Deserialize(serialized, initial []byte, opts ...Option) (*BlockDevice, error) {
	bd, err := New(initial, opts...)
	if err != nil {
		return nil, err
	}

	var start time.Time
	if bd.cfg.observer != nil {
		start = time.Now()
	}
	loadErr := bd.loadOverlay(serialized)
	if bd.cfg.observer != nil {
		bd.fireObserver(Event{
			Op:       OpDeserialize,
			Device:   bd.cfg.name,
			Length:   len(serialized),
			Blocks:   len(serialized) / EntrySize,
			Duration: time.Since(start),
			Err:      loadErr,
		})
	}
	if loadErr != nil {
		return nil, loadErr
	}
	return bd, nil
}

func (b *BlockDevice) loadOverlay(serialized []byte) error {
	if len(serialized) == 0 {
		return nil
	}
	if len(serialized)%EntrySize != 0 {
		return fmt.Errorf("blockdev.Deserialize: len(serialized)=%d not a multiple of %d: %w",
			len(serialized), EntrySize, ErrBadFormat)
	}
	maxBlock := b.length / BlockSize
	nEntries := len(serialized) / EntrySize
	prevBlock := int64(-1)
	for i := 0; i < nEntries; i++ {
		off := i * EntrySize
		blockNum := int64(binary.BigEndian.Uint64(serialized[off:]))
		if blockNum < 0 || blockNum >= maxBlock {
			return fmt.Errorf("blockdev.Deserialize: entry %d: block#=%d out of range [0, %d): %w",
				i, blockNum, maxBlock, ErrBadFormat)
		}
		if blockNum <= prevBlock {
			return fmt.Errorf("blockdev.Deserialize: entry %d: block#=%d not strictly ascending after %d: %w",
				i, blockNum, prevBlock, ErrBadFormat)
		}
		prevBlock = blockNum
		b.ov.put(blockNum, serialized[off+8:off+EntrySize])
	}
	return nil
}
