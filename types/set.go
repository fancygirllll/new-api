package types

type Set[T comparable] struct {
	items map[T]struct{}
}

// NewSet creates and returns a new Set
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		items: make(map[T]struct{}),
	}
}

func (s *Set[T]) Add(item T) {
	s.items[item] = struct{}{}
}

// Remove removes an element from the Set
func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

// Contains checks whether the Set contains an element
func (s *Set[T]) Contains(item T) bool {
	_, exists := s.items[item]
	return exists
}

// Len returns the number of elements in the Set
func (s *Set[T]) Len() int {
	return len(s.items)
}

// Items returns a slice of all elements in the Set
// Note: because maps are unordered, the returned slice order is random
func (s *Set[T]) Items() []T {
	items := make([]T, 0, s.Len())
	for item := range s.items {
		items = append(items, item)
	}
	return items
}
