package blockdev

import (
	"fmt"
	"io"
	"time"

	"github.com/codingminions/blockdev/internal/overlay"
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
	base    []byte
	length  int64
	overlay *overlay.Store
	cfg     config
}

var (
	_ io.ReaderAt = (*BlockDevice)(nil)
	_ io.WriterAt = (*BlockDevice)(nil)
)

// New validates alignment up front rather than at first ReadAt — a misaligned
// base is a configuration bug, and we'd rather fail at construction than after
// a sandbox has been wired up.
func New(initial []byte, opts ...Option) (*BlockDevice, error) {
	if err := validate.Alignment(0, len(initial)); err != nil {
		return nil, fmt.Errorf("blockdev.New: len(initial)=%d: %w", len(initial), ErrMisaligned)
	}
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	return &BlockDevice{
		base:    initial,
		length:  int64(len(initial)),
		overlay: overlay.New(),
		cfg:     cfg,
	}, nil
}

// ReadAt validates bounds before alignment so negative offsets surface as
// ErrOutOfBounds (a range bug) rather than ErrMisaligned (a shape bug); the
// distinction matters for log greppability when NBD pipelines requests.
// On any failure returns (0, err) and leaves p untouched — partial reads
// could corrupt the guest filesystem before the kernel notices.
func (b *BlockDevice) ReadAt(p []byte, off int64) (int, error) {
	var start time.Time
	if b.cfg.observer != nil {
		start = time.Now()
	}
	n, err := b.readAt(p, off)
	if b.cfg.observer != nil {
		b.fireObserver(Event{
			Op:       OpRead,
			Device:   b.cfg.name,
			Offset:   off,
			Length:   len(p),
			Blocks:   len(p) / BlockSize,
			Duration: time.Since(start),
			Err:      err,
		})
	}
	return n, err
}

func (b *BlockDevice) readAt(p []byte, off int64) (int, error) {
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
	for i := 0; i < len(p); i += BlockSize {
		blockOff := off + int64(i)
		blockNum := blockOff / BlockSize
		if data, ok := b.overlay.Get(blockNum); ok {
			copy(p[i:i+BlockSize], data)
		} else {
			copy(p[i:i+BlockSize], b.base[blockOff:blockOff+BlockSize])
		}
	}
	return len(p), nil
}

// WriteAt captures writes into the overlay; the base is never mutated. Caller
// may reuse p immediately — the overlay copies what it stores.
func (b *BlockDevice) WriteAt(p []byte, off int64) (int, error) {
	var start time.Time
	if b.cfg.observer != nil {
		start = time.Now()
	}
	n, err := b.writeAt(p, off)
	if b.cfg.observer != nil {
		b.fireObserver(Event{
			Op:       OpWrite,
			Device:   b.cfg.name,
			Offset:   off,
			Length:   len(p),
			Blocks:   len(p) / BlockSize,
			Duration: time.Since(start),
			Err:      err,
		})
	}
	return n, err
}

func (b *BlockDevice) writeAt(p []byte, off int64) (int, error) {
	if err := validate.Bounds(off, len(p), b.length); err != nil {
		return 0, fmt.Errorf("blockdev.WriteAt off=%d len=%d device=%d: %w",
			off, len(p), b.length, ErrOutOfBounds)
	}
	if err := validate.Alignment(off, len(p)); err != nil {
		return 0, fmt.Errorf("blockdev.WriteAt off=%d len=%d: %w",
			off, len(p), ErrMisaligned)
	}
	if len(p) == 0 {
		return 0, nil
	}
	for i := 0; i < len(p); i += BlockSize {
		blockNum := (off + int64(i)) / BlockSize
		b.overlay.Put(blockNum, p[i:i+BlockSize])
	}
	return len(p), nil
}

// Isolated so the deferred recover catches only the observer's panic, never a
// panic that originated in the I/O path.
func (b *BlockDevice) fireObserver(e Event) {
	defer func() { _ = recover() }()
	b.cfg.observer(e)
}
