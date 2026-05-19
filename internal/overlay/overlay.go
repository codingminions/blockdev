// Package overlay is the copy-on-write block store: reads expose the stored
// slice (callers must not mutate), Puts copy so callers can reuse their buffer.
package overlay

import "sync"

type Store struct {
	mu sync.RWMutex
	m  map[int64][]byte
}

func New() *Store {
	return &Store{m: make(map[int64][]byte)}
}

// Returned slice aliases the store — caller must not mutate.
func (s *Store) Get(block int64) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.m[block]
	return data, ok
}

// Copies data so the caller can reuse its buffer immediately.
func (s *Store) Put(block int64, data []byte) {
	buf := make([]byte, len(data))
	copy(buf, data)
	s.mu.Lock()
	s.m[block] = buf
	s.mu.Unlock()
}

// fn must not mutate data; return false to stop early.
func (s *Store) Range(fn func(block int64, data []byte) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for block, data := range s.m {
		if !fn(block, data) {
			return
		}
	}
}

func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
