// Package stream provides a way to construct data processing streams from smaller pieces.
package stream

import (
	"log"

	"github.com/rprtr258/vk-utils/flow/fun"
	"github.com/rprtr258/vk-utils/flow/slice"
)

// Stream is a finite or infinite stream of values.
type Stream[A any] interface {
	// Next gives either value or nothing if stream has ended.
	Next() fun.Option[A]
}

type mapImpl[A, B any] struct {
	Stream[A]
	f func(A) B
}

func (xs *mapImpl[A, B]) Next() fun.Option[B] {
	return fun.FoldOption(
		xs.Stream.Next(),
		func(a A) fun.Option[B] { return fun.Some(xs.f(a)) },
		fun.None[B],
	)
}

// Map converts values of the stream.
func Map[A any, B any](xs Stream[A], f func(A) B) Stream[B] {
	return &mapImpl[A, B]{xs, f}
}

type chainImpl[A any] struct {
	first  Stream[A]
	second Stream[A]
}

func (xs *chainImpl[A]) Next() fun.Option[A] {
	return fun.FoldOption(
		xs.first.Next(),
		fun.Some[A],
		func() fun.Option[A] { return xs.second.Next() },
	)
}

// Chain appends another stream after the end of the first one.
func Chain[A any](as Stream[A], bs Stream[A]) Stream[A] {
	return &chainImpl[A]{as, bs}
}

type flatMapImpl[A, B any] struct {
	Stream[A]
	f    func(A) Stream[B]
	last Stream[B]
}

func (xs *flatMapImpl[A, B]) Next() fun.Option[B] {
	y := xs.last.Next()
	if y.IsNone() {
		xs.last = fun.FoldOption(fun.Map(xs.Stream.Next(), xs.f), fun.Identity[Stream[B]], NewStreamEmpty[B])
		y = xs.last.Next()
	}
	return y
}

func FlatMap[A any, B any](xs Stream[A], f func(A) Stream[B]) Stream[B] {
	return &flatMapImpl[A, B]{xs, f, NewStreamEmpty[B]()}
}

// Flatten simplifies a stream of streams to just the stream of values by concatenating all inner streams.
func Flatten[A any](xs Stream[Stream[A]]) Stream[A] {
	return FlatMap(xs, fun.Identity[Stream[A]])
}

func Sum[A slice.Number](xs Stream[A]) A {
	var zero A
	return Reduce(zero,
		func(x A, y A) A {
			return x + y
		},
		xs,
	)
}

type chunkedImpl[A any] struct {
	Stream[A]
	chunkSize int
}

func (xs *chunkedImpl[A]) Next() fun.Option[[]A] {
	x := xs.Stream.Next()
	if x.IsNone() {
		return fun.None[[]A]()
	}
	chunk := make([]A, 1, xs.chunkSize)
	chunk[0] = x.Unwrap()
	for i := 1; i < xs.chunkSize; i++ {
		x := xs.Stream.Next()
		if x.IsNone() {
			break
		}
		chunk = append(chunk, x.Unwrap())
	}
	return fun.Some(chunk)
}

// Chunked groups elements by n and produces a stream of slices.
func Chunked[A any](xs Stream[A], n int) Stream[[]A] {
	return &chunkedImpl[A]{xs, n}
}

type intersperseImpl[A any] struct {
	Stream[A]
	sep       A
	isSepNext bool
}

func (xs *intersperseImpl[A]) Next() fun.Option[A] {
	if xs.isSepNext {
		xs.isSepNext = false
		return fun.Some(xs.sep)
	}
	x := xs.Stream.Next()
	if x.IsNone() {
		return x
	}
	xs.isSepNext = true
	return x
}

// Intersperse adds a separator after each stream element
func Intersperse[A any](xs Stream[A], sep A) Stream[A] {
	return &intersperseImpl[A]{xs, sep, false}
}

type repeatImpl[A any] struct {
	Stream[A]
	i   int
	buf []A
}

func (xs *repeatImpl[A]) Next() fun.Option[A] {
	x := xs.Stream.Next()
	if x.IsNone() {
		res := xs.buf[xs.i]
		xs.i = (xs.i + 1) % len(xs.buf)
		return fun.Some(res)
	}
	xs.buf = append(xs.buf, x.Unwrap())
	return x
}

// Repeat appends the same stream infinitely.
func Repeat[A any](xs Stream[A]) Stream[A] {
	return &repeatImpl[A]{xs, 0, make([]A, 0)}
}

type takeImpl[A any] struct {
	Stream[A]
	n uint
}

func (xs *takeImpl[A]) Next() fun.Option[A] {
	if xs.n == 0 {
		return fun.None[A]()
	}
	xs.n--
	return xs.Stream.Next()
}

// Take cuts the stream after n elements.
func Take[A any](xs Stream[A], n uint) Stream[A] {
	return &takeImpl[A]{xs, n}
}

// Skip skips n elements in the stream.
func Skip[A any](xs Stream[A], n int) Stream[A] {
	for i := 0; i < n; i++ {
		if x := xs.Next(); x.IsNone() {
			break
		}
	}
	return xs
}

type filterImpl[A any] struct {
	Stream[A]
	p func(A) bool
}

func (xs *filterImpl[A]) Next() fun.Option[A] {
	for {
		x := xs.Stream.Next()
		if x.IsNone() {
			break
		}
		if xs.p(x.Unwrap()) {
			return x
		}
	}
	return fun.None[A]()
}

// Filter leaves in the stream only the elements that satisfy the given predicate.
func Filter[A any](xs Stream[A], p func(A) bool) Stream[A] {
	return &filterImpl[A]{xs, p}
}

func Find[A any](xs Stream[A], p func(A) bool) fun.Option[A] {
	return Filter(xs, p).Next()
}

type takeWhileImpl[A any] struct {
	Stream[A]
	p     func(A) bool
	ended bool
}

func (xs *takeWhileImpl[A]) Next() fun.Option[A] {
	if xs.ended {
		return fun.None[A]()
	}
	if x := xs.Stream.Next(); x.IsSome() && xs.p(x.Unwrap()) {
		return x
	}
	xs.ended = true
	return fun.None[A]()
}

func TakeWhile[A any](xs Stream[A], p func(A) bool) Stream[A] {
	return &takeWhileImpl[A]{xs, p, false}
}

// DebugPrint prints every processed element, without changing it.
func DebugPrint[A any](prefix string, xs Stream[A]) Stream[A] {
	return Map(xs, func(a A) A {
		log.Println(prefix, " ", a)
		return a
	})
}

type uniqueImpl[A comparable] struct {
	Stream[A]
	seen fun.Set[A]
}

func (xs *uniqueImpl[A]) Next() fun.Option[A] {
	for {
		x := xs.Stream.Next()
		if x.IsNone() {
			return fun.None[A]()
		}
		xVal := x.Unwrap()
		if !xs.seen.Contains(xVal) {
			xs.seen[xVal] = fun.Unit1
			return x
		}
	}
}

func Unique[A comparable](xs Stream[A]) Stream[A] {
	return &uniqueImpl[A]{xs, make(fun.Set[A])}
}
