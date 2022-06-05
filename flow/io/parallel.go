package io

// import (
// 	"github.com/rprtr258/goflow/fun"
// 	"github.com/rprtr258/goflow/slice"
// )

// // Parallel starts the given IOs in Go routines and waits for all results
// func Parallel[A any](ios []IO[A]) IO[[]A] {
// 	ioFibers := slice.Map(ios, Start[A])
// 	fibersIO := Sequence(ioFibers)
// 	return FlatMap(fibersIO, func(fibers []Fiber[A]) IO[[]A] {
// 		joins := slice.Map(fibers, func(fiber Fiber[A]) IO[A] { return fiber.Join() })
// 		return Sequence(joins)
// 	})
// }

// // ConcurrentlyFirst - runs all IOs in parallel.
// // returns the very first result.
// // TODO: after obtaining result - cancel the other IOs.
// func ConcurrentlyFirst[A any](ios []IO[A]) IO[A] {
// 	channelIO := Pure(func() chan GoResult[A] {
// 		return make(chan GoResult[A], len(ios))
// 		// we will only read the very first response. Hence the other go routines could hang if sending to unbuffered channel
// 	})
// 	return FlatMap(channelIO, func(channel chan GoResult[A]) IO[A] {
// 		ioSendToChannel := slice.Map(ios, func(ioa IO[A]) IO[fun.Unit] {
// 			goResult := FoldToGoResult(ioa)
// 			return FlatMap(goResult, ToChannel(channel))
// 		})
// 		parallelSendResults := Parallel(ioSendToChannel)
// 		ignoreParallelResultButCloseChannelAfterwards := FireAndForget(AndThen(parallelSendResults, CloseChannel(channel)))
// 		readFirstFromChannel := FromChannel(channel)
// 		ignoreParallelResultsAndThenReadFirstFromChannel := AndThen(
// 			ignoreParallelResultButCloseChannelAfterwards,
// 			readFirstFromChannel)
// 		return UnfoldGoResult(ignoreParallelResultsAndThenReadFirstFromChannel)
// 	})
// }
