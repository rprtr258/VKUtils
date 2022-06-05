package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestStream(t *testing.T) {
// 	empty := Empty[int]()
// 	_, err := io.UnsafeRunSync(DrainAll(empty))
// 	assert.Equal(t, nil, err)
// 	stream10_12 := LiftMany(10, 11, 12)
// 	stream20_24 := Mul2(stream10_12)
// 	res, err := io.UnsafeRunSync(ToSlice(stream20_24))
// 	assert.Equal(t, nil, err)
// 	assert.Equal(t, []int{20, 22, 24}, res)
// }

// func TestGenerate(t *testing.T) {
// 	powers2 := Unfold(1, func(s int) int {
// 		return s * 2
// 	})

// 	res, err := io.UnsafeRunSync(Head(powers2))
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, res)

// 	powers2_10 := Drop(powers2, 9)
// 	res, err = io.UnsafeRunSync(Head(powers2_10))
// 	assert.NoError(t, err)
// 	assert.Equal(t, 1024, res)
// }

// func TestDrainAll(t *testing.T) {
// 	results := []int{}
// 	natsAppend := MapEval(
// 		Take(Repeat(nats10), 10),
// 		func(a int) io.IO[int] {
// 			return io.Eval(func() (int, error) {
// 				results = append(results, a)
// 				return a, nil
// 			})
// 		})
// 	_, err := io.UnsafeRunSync(DrainAll(natsAppend))
// 	assert.NoError(t, err)
// 	assert.ElementsMatch(t, results, nats10Values)
// }

func TestSum(t *testing.T) {
	sum := Sum(nats10)
	assert.Equal(t, 45, sum)
}

// func isEven(i int) bool {
// 	return i%2 == 0
// }

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

// 	filtered := Filter(natsRepeated, isEven)
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
