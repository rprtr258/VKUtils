package stream

import (
	"testing"
	"time"

	"github.com/rprtr258/vk-utils/flow/slice"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()
	sleepTasks := Map(nats(), func(id int) func() int {
		return func() int {
			time.Sleep(10 * time.Millisecond)
			return id
		}
	})
	sleepTasks100 := Take(sleepTasks, 100)
	pool := NewPool[int](10)
	resultStream := pool(sleepTasks100)
	results := CollectToSlice(resultStream)
	start := time.Now()
	assert.Equal(t, 100, slice.SetSize(slice.ToSet(results)))
	assert.WithinDuration(t, start, time.Now(), 150*time.Millisecond)
}
