package stream

import (
	// 	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func nats10() Stream[int] {
	return Take(Generate(0, func(s int) int { return s + 1 }), 10)
}

var mul2 = func(i int) int { return i * 2 }

// var pipeMul2IO = PipeToPairOfChannels(mul2)

// var printInt = NewSink(func(i int) { fmt.Printf("%d", i) })

func TestStream(t *testing.T) {
	empty := NewStreamEmpty[int]()
	DrainAll(empty)

	res := CollectToSlice(Map(LiftMany(10, 11, 12), mul2))
	assert.Equal(t, []int{20, 22, 24}, res)
}

func TestGenerate(t *testing.T) {
	powers2 := Generate(1, mul2)

	res := Head(powers2).Unwrap()
	assert.Equal(t, 1, res)

	res = Head(Drop(powers2, 9)).Unwrap()
	assert.Equal(t, 1024, res)
}

func TestDrainAll(t *testing.T) {
	results := CollectToSlice(Take(Repeat(nats10()), 10))
	assert.ElementsMatch(t, results, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
}

func TestSum(t *testing.T) {
	sum := Sum(nats10())
	assert.Equal(t, 45, sum)
}

// func TestFlatMapPipe(t *testing.T) {
// 	natsRepeated := FlatMapPipe(func(i int) Stream[int] {
// 		return MapPipe(func(j int) int {
// 			return i + j
// 		})(nats10)
// 	})(nats10)

// 	ioLen := Head(Len(natsRepeated))
// 	len, err := io.UnsafeRunSync(ioLen)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 100, len)

// 	filtered := Filter(natsRepeated, func(i int) bool {
// 	return i%2 == 0
// })
// 	sumStream := Sum(filtered)
// 	ioSum := Head(sumStream)
// 	var sum int
// 	sum, err = io.UnsafeRunSync(ioSum)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 550, sum)
// }

// func TestChunks(t *testing.T) {
// 	natsBy10 := ChunkN[int](10)(Take(nats, 19))
// 	nats10to19IO := Head(Drop(natsBy10, 1))
// 	nats10to19, err := io.UnsafeRunSync(nats10to19IO)
// 	assert.NoError(t, err)
// 	assert.ElementsMatch(t, []int{11, 12, 13, 14, 15, 16, 17, 18, 19}, nats10to19)
// }

// func TestForEach(t *testing.T) {
// 	powers2 := Generate(1, mul2)
// 	is := []int{}
// 	forEachIO := ForEach(Take(powers2, 5), func(i int) {
// 		is = append(is, i)
// 	})
// 	_, err := io.UnsafeRunSync(forEachIO)
// 	assert.NoError(t, err)
// 	assert.ElementsMatch(t, []int{2, 4, 8, 16, 32}, is)
// }
