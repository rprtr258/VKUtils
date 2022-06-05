package stream

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	t.Parallel()
	sleepTasks := MapToTasks(
		nats(),
		func(id int) int {
			time.Sleep(10 * time.Millisecond)
			return id
		},
	)
	sleepTasks100 := Take(sleepTasks, 100)
	pool := NewPool[int](10)
	resultStream := Unique(pool(sleepTasks100))
	start := time.Now()
	assert.Equal(t, 100, Count(resultStream))
	assert.WithinDuration(t, start, time.Now(), 150*time.Millisecond)
}
