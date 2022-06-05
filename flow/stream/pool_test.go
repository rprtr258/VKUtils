package stream

// import (
// 	"testing"
// 	"time"

// 	"github.com/rprtr258/goflow/io"
// 	"github.com/rprtr258/goflow/slice"
// 	"github.com/stretchr/testify/assert"
// )

// func TestPool(t *testing.T) {
// 	sleep10ms := func(id int) io.IO[int] {
// 		return io.Pure(func() int {
// 			time.Sleep(10 * time.Millisecond)
// 			return id
// 		})
// 	}
// 	sleepTasks := Map(nats, sleep10ms)
// 	sleepTasks100 := Take(sleepTasks, 100)

// 	poolIO := NewPool[int](10)

// 	resultsIO := io.FlatMap(poolIO, func(pool Pool[int]) io.IO[[]int] {
// 		sleepResults := ThroughPool(sleepTasks100, pool)
// 		resultStream := MapEval(sleepResults, io.FromConstantGoResult[int])
// 		return ToSlice(resultStream)
// 	})
// 	start := time.Now()
// 	results, err := io.UnsafeRunSync(resultsIO)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 100, slice.SetSize(slice.ToSet(results)))
// 	assert.WithinDuration(t, start, time.Now(), 200*time.Millisecond)
// }
