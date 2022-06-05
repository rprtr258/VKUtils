package fun

// Set is a collection of distinct elements.
type Set[A comparable] map[A]Unit

// Contains checks if element is in set.
func (s *Set[A]) Contains(a A) bool {
	_, ok := (*s)[a]
	return ok
}
