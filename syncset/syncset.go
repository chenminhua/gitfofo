package syncset

import "sync"

// Set is a thread-safe set of values of type T.
type Set[T comparable] struct {
	mu sync.RWMutex
	m  map[T]struct{}
}

// New returns an initialized Set.
func New[T comparable]() *Set[T] {
	return &Set[T]{m: make(map[T]struct{})}
}

// Add inserts val into the set.
func (s *Set[T]) Add(val T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[val] = struct{}{}
}

// Remove deletes val from the set.
func (s *Set[T]) Remove(val T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, val)
}

// Contains reports whether val is in the set.
func (s *Set[T]) Contains(val T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.m[val]
	return ok
}

// Snapshot returns a regular map[T]struct{} containing all elements,
// safe for iteration without any further locking.
func (s *Set[T]) Snapshot() map[T]struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := make(map[T]struct{}, len(s.m))
	for k := range s.m {
		snap[k] = struct{}{}
	}
	return snap
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
