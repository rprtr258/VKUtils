package stream

// import (
// 	"fmt"
// )

var nats = Generate(0, func(s int) int {
	return s + 1
})

var nats10 = Take(nats, 10)

// var nats10Values = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// var Mul2 = MapPipe(func(i int) int { return i * 2 })
// var pipeMul2IO = PipeToPairOfChannels(Mul2)

// var printInt = NewSink(func(i int) { fmt.Printf("%d", i) })
