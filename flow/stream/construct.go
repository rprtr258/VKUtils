// functions to make Stream from something that is not stream
package stream

import (
	"github.com/rprtr258/vk-utils/flow/fun"
)

// var empty = Empty[fun.Unit]()

// // EmptyUnit returns an empty stream of units.
// // It's more performant because the same instance is being used.
// func EmptyUnit() Stream[fun.Unit] {
// 	return empty
// }

// // Eval returns a stream of one value that is the result of IO.
// func Eval[A any](ioa io.IO[A]) Stream[A] {
// 	return io.Map(ioa, func(a A) StepResult[A] {
// 		return NewStepResult(a, Empty[A]())
// 	})
// }

// // Lift returns a stream of one value.
// func Lift[A any](a A) Stream[A] {
// 	return Eval(io.Lift(a))
// }

// LiftMany returns a stream with all the given values.
func LiftMany[A any](as ...A) Stream[A] {
	return FromSlice(as)
}

type sliceImpl[A any] struct {
	data []A
	i    int
}

func (xs *sliceImpl[A]) Next() fun.Option[A] {
	if xs.i == len(xs.data) {
		return fun.None[A]()
	}
	xs.i++
	return fun.Some(xs.data[xs.i-1])
}

// FromSlice constructs a stream from the slice.
func FromSlice[A any](xs []A) Stream[A] {
	return &sliceImpl[A]{xs, 0}
}

func FromSet[A comparable](xs fun.Set[A]) Stream[A] {
	slice := make([]A, 0, len(xs))
	for k := range xs {
		slice = append(slice, k)
	}
	return FromSlice(slice)
}

type generateImpl[A any] struct {
	x A
	f func(A) A
}

func (xs *generateImpl[A]) Next() fun.Option[A] {
	res := xs.x
	xs.x = xs.f(xs.x)
	return fun.Some(res)
}

// Generate constructs an infinite stream of values using the production function.
func Generate[A any](x0 A, f func(A) A) Stream[A] {
	return &generateImpl[A]{x0, f}
}

// // FromSideEffectfulFunction constructs a stream from a Go-style function.
// // It is expected that this function is not pure and can return different results.
// func FromSideEffectfulFunction[A any](f func() (A, error)) Stream[A] {
// 	return Repeat(Eval(io.Eval(f)))
// }

// // FromStepResult constructs a stream from an IO that returns StepResult.
// func FromStepResult[A any](iosr io.IO[StepResult[A]]) Stream[A] {
// 	return iosr
// }

type emptyImpl[A any] struct{}

func (xs *emptyImpl[A]) Next() fun.Option[A] {
	return fun.None[A]()
}

// NewStreamEmpty returns an empty stream.
func NewStreamEmpty[A any]() Stream[A] {
	return &emptyImpl[A]{}
}

// TODO: fix blocking
type chanStream[A any] <-chan A

func (s chanStream[A]) Next() fun.Option[A] {
	val, isClosed := <-s
	if isClosed {
		return fun.None[A]()
	}
	return fun.Some(val)
}
