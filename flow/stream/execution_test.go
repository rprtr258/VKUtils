package stream

// import (
// 	"testing"

// 	"github.com/rprtr258/goflow/io"
// 	"github.com/stretchr/testify/assert"
// )

// func TestForEach(t *testing.T) {
// 	powers2 := Unfold(1, func(s int) int {
// 		return s * 2
// 	})
// 	is := []int{}
// 	forEachIO := ForEach(Take(powers2, 5), func(i int) {
// 		is = append(is, i)
// 	})
// 	_, err := io.UnsafeRunSync(forEachIO)
// 	assert.NoError(t, err)
// 	assert.ElementsMatch(t, []int{2, 4, 8, 16, 32}, is)
// }
