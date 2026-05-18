package blockdev

import (
	"fmt"
	"io"

	"github.com/codingminions/blockdev/internal/validate"
)

// BlockSize is 4096 to match the OS page size and NBD's expected sector size.
// Anything smaller would cost read-modify-write; anything larger would inflate
// the overlay's per-changed-block storage.
const BlockSize = 4096

// BlockDevice retains the base without copying. Caller must not mutate base
// after handoff; defensive copy would double peak memory when many sandboxes
// share one image.
type BlockDevice struct {
	base   []byte
	length int64
}

var _ io.ReaderAt = (*BlockDevice)(nil)

// New validates alignment up front rather than at first ReadAt — a misaligned
// base is a configuration bug, and we'd rather fail at construction than after
// a sandbox has been wired up.
func New(initial []byte) (*BlockDevice, error) {
	if err := validate.Alignment(0, len(initial)); err != nil {
		return nil, fmt.Errorf("blockdev.New: len(initial)=%d: %w", len(initial), ErrMisaligned)
	}
	return &BlockDevice{
		base:   initial,
		length: int64(len(initial)),
	}, nil
}

// ReadAt validates bounds before alignment so negative offsets surface as
// ErrOutOfBounds (a range bug) rather than ErrMisaligned (a shape bug); the
// distinction matters for log greppability when NBD pipelines requests.
// On any failure returns (0, err) and leaves p untouched — partial reads
// could corrupt the guest filesystem before the kernel notices.
func (b *BlockDevice) ReadAt(p []byte, off int64) (int, error) {
	if err := validate.Bounds(off, len(p), b.length); err != nil {
		return 0, fmt.Errorf("blockdev.ReadAt off=%d len=%d device=%d: %w",
			off, len(p), b.length, ErrOutOfBounds)
	}
	if err := validate.Alignment(off, len(p)); err != nil {
		return 0, fmt.Errorf("blockdev.ReadAt off=%d len=%d: %w",
			off, len(p), ErrMisaligned)
	}
	if len(p) == 0 {
		return 0, nil
	}
	copy(p, b.base[off:off+int64(len(p))])
	return len(p), nil
}
