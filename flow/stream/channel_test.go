package stream

import (
	"testing"

	// 	"github.com/rprtr258/goflow/fun"

	"github.com/stretchr/testify/assert"
)

func TestSendingDataThroughChannel(t *testing.T) {
	ch := make(chan int)
	results := CollectToSlice(FromPairOfChannels(nats10(), ch, ch))
	assert.ElementsMatch(t, results, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
}

// func TestStreamConversion(t *testing.T) {
// 	io2 := io.ForEach(pipeMul2IO, func(pair fun.Pair[chan<- int, <-chan int]) {
// 		input := pair.Left
// 		output := pair.Right
// 		input <- 10
// 		o := <-output
// 		assert.Equal(t, 20, o)
// 	})
// 	_, err := io.UnsafeRunSync(io2)
// 	assert.NoError(t, err)
// }
