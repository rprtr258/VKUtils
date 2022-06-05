package stream

// functions to make something from Stream that is not Stream.

import (
	"github.com/rprtr258/vk-utils/flow/fun"
)

// ForEach invokes a simple function for each element of the stream.
func ForEach[A any](xs Stream[A], f func(A)) {
	for x := xs.Next(); x.IsSome(); x = xs.Next() {
		f(x.Unwrap())
	}
}

// DrainAll throws away all values.
func DrainAll[A any](xs Stream[A]) {
	ForEach(xs, func(_ A) {})
}

// CollectToSlice executes the stream and collects all results to a slice.
func CollectToSlice[A any](xs Stream[A]) []A {
	slice := make([]A, 0)
	ForEach(xs, func(a A) { slice = append(slice, a) })
	return slice
}

// CollectToSet executes the stream and collects all results to a set.
func CollectToSet[A comparable](xs Stream[A]) fun.Set[A] {
	set := make(fun.Set[A])
	ForEach(xs, func(a A) { set[a] = fun.Unit1 })
	return set
}

// Head takes the first element if present.
func Head[A any](xs Stream[A]) fun.Option[A] {
	return xs.Next()
}

// Reduce reduces stream into one value using given operation.
func Reduce[A, B any](start A, op func(A, B) A, xs Stream[B]) A {
	for x := xs.Next(); x.IsSome(); x = xs.Next() {
		start = op(start, x.Unwrap())
	}
	return start
}

// Count consumes stream and returns it's length.
func Count[A any](xs Stream[A]) int {
	return Sum(Map(xs, fun.Const[A](1)))
}

// Group groups elements by a function that returns a key.
func Group[A any, K comparable](xs Stream[A], by func(A) K) map[K][]A {
	res := make(map[K][]A)
	ForEach(
		xs,
		func(a A) {
			key := by(a)
			vals, ok := res[key]
			if ok {
				vals = append(vals, a)
				res[key] = vals
			} else {
				res[key] = []A{a}
			}
		},
	)
	return res
}

// GroupMap is a convenience function that groups and then maps the subslices.
func GroupMap[A, B any, K comparable](xs Stream[A], by func(A) K, aggregate func([]A) B) map[K]B {
	tmp := Group(xs, by)
	res := make(map[K]B, len(tmp))
	for k, v := range tmp {
		res[k] = aggregate(v)
	}
	return res
}

// GroupCount for each key counts how often it is seen.
func GroupCount[A any, K comparable](xs Stream[A], by func(A) K) map[K]int {
	return GroupMap(xs, by, func(ys []A) int { return len(ys) })
}
