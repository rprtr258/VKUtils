package stream

// import (
// 	"testing"

// 	"github.com/rprtr258/goflow/fun"
// 	"github.com/rprtr258/goflow/io"
// 	"github.com/stretchr/testify/assert"
// )

// func TestSendingDataThroughChannel(t *testing.T) {
// 	ch := make(chan int)
// 	pipe := PairOfChannelsToPipe(ch, ch)
// 	nats10After := Through(nats10, pipe)
// 	results, err := io.UnsafeRunSync(ToSlice(nats10After))
// 	assert.NoError(t, err)
// 	assert.ElementsMatch(t, results, nats10Values)
// }

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
