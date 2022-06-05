package stream

// func TestPool(t *testing.T) {
// 	sleepTasks := Map(nats(), func(id int) func() int {
// 		return func() int {
// 			time.Sleep(10 * time.Millisecond)
// 			return id
// 		}
// 	})
// 	sleepTasks100 := Take(sleepTasks, 100)
// 	pool := NewPool[func() int](10)
// 	sleepResults := pool(sleepTasks100)
// 	resultStream := Map(sleepResults, func(f func() int) int { return f() })
// 	results := CollectToSlice(resultStream)
// 	start := time.Now()
// 	assert.Equal(t, 100, slice.SetSize(slice.ToSet(results)))
// 	assert.WithinDuration(t, start, time.Now(), 200*time.Millisecond)
// }
