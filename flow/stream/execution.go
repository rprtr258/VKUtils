// functions to make something from Stream that is not Stream
package stream

import (
	// 	"fmt"

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

// Head takes the first element if present.
func Head[A any](xs Stream[A]) fun.Option[A] {
	return xs.Next()
}

func Reduce[A, B any](start A, op func(A, B) A, xs Stream[B]) A {
	for x := xs.Next(); x.IsSome(); x = xs.Next() {
		start = op(start, x.Unwrap())
	}
	return start
}
