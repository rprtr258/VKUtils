package stream

type Pipe[A, B any] func(Stream[A]) Stream[B]
