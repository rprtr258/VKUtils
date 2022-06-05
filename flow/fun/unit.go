package fun

// Unit is a type that has only a single value.
type Unit struct{}

// Unit1 is the value of type Unit.
var Unit1 = Unit{}

// Set is a collection of distinct elements
type Set[A comparable] map[A]Unit

// Contains checks if element is in set
func (s *Set[A]) Contains(a A) bool {
	_, ok := (*s)[a]
	return ok
}
