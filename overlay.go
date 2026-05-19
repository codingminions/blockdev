package blockdev

import "sync"

// overlay is the copy-on-write block store: reads expose the stored slice
// (callers must not mutate), puts copy so callers can reuse their buffer.
type overlay struct {
	mu sync.RWMutex
	m  map[int64][]byte
}

func newOverlay() *overlay {
	return &overlay{m: make(map[int64][]byte)}
}

// Returned slice aliases the store — caller must not mutate.
func (o *overlay) get(block int64) ([]byte, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	data, ok := o.m[block]
	return data, ok
}

// Copies data so the caller can reuse its buffer immediately.
func (o *overlay) put(block int64, data []byte) {
	buf := make([]byte, len(data))
	copy(buf, data)
	o.mu.Lock()
	o.m[block] = buf
	o.mu.Unlock()
}

// fn must not mutate data; return false to stop early.
func (o *overlay) each(fn func(block int64, data []byte) bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	for block, data := range o.m {
		if !fn(block, data) {
			return
		}
	}
}

func (o *overlay) count() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.m)
}
