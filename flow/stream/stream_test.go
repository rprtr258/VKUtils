package stream

import (
	// 	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func nats() Stream[int] {
	return Generate(0, func(s int) int { return s + 1 })
}

func nats10() Stream[int] {
	return Take(nats(), 10)
}

var mul2 = func(i int) int { return i * 2 }

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

func TestFlatMapPipe(t *testing.T) {
	pipe := FlatMapPipe(func(i int) Stream[int] {
		return Map(nats10(), func(j int) int {
			return i + j
		})
	})
	sum := Sum(Filter(pipe(nats10()), func(i int) bool {
		return i%2 == 0
	}))
	assert.Equal(t, 450, sum)
}

func TestChunks(t *testing.T) {
	natsBy10 := Chunked(Take(nats(), 19), 10)
	nats10to19 := Head(Drop(natsBy10, 1)).Unwrap()
	assert.ElementsMatch(t, []int{10, 11, 12, 13, 14, 15, 16, 17, 18}, nats10to19)
}

func TestForEach(t *testing.T) {
	powers2 := Generate(1, mul2)
	is := []int{}
	ForEach(Take(powers2, 5), func(i int) {
		is = append(is, i)
	})
	assert.ElementsMatch(t, []int{1, 2, 4, 8, 16}, is)
}
