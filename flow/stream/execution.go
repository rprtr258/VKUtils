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

// // DrainAll executes the stream and throws away all values.
// func DrainAll[A any](stm Stream[A]) io.IO[fun.Unit] {
// 	return io.FlatMap[StepResult[A]](stm, func(sra StepResult[A]) io.IO[fun.Unit] {
// 		if sra.IsFinished {
// 			return io.Lift(fun.Unit1)
// 		} else {
// 			return DrainAll(sra.Continuation)
// 		}
// 	})
// }

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
