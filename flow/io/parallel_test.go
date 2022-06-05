package io

// import (
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// )

// func TestParallel(t *testing.T) {
// 	start := time.Now()
// 	sleep100ms := SleepA(100*time.Millisecond, "a")
// 	ios := []IO[string]{}
// 	for i := 0; i < 100; i += 1 {
// 		ios = append(ios, sleep100ms)
// 	}
// 	ioall := Parallel(ios)
// 	results, err := UnsafeRunSync(ioall)
// 	assert.Equal(t, err, nil)
// 	end := time.Now()
// 	assert.Equal(t, results[0], "a")
// 	assert.WithinDuration(t, end, start, 200*time.Millisecond)
// }
