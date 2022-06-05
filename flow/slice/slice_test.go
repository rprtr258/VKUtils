package slice

// import (
// 	"testing"

// 	"github.com/rprtr258/vk-utils/flow/fun"
// 	"github.com/stretchr/testify/assert"
// )

// func IsEven(i int) bool {
// 	return i%2 == 0
// }

// var nats10Values = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// func TestFilter(t *testing.T) {
// 	t.Parallel()
// 	sumEven := Sum(Filter(nats10Values, IsEven))
// 	assert.Equal(t, 30, sumEven)

// 	sumOdd := Sum(FilterNot(nats10Values, IsEven))
// 	assert.Equal(t, 25, sumOdd)
// }

// func TestFlatten(t *testing.T) {
// 	t.Parallel()
// 	floatsNested := Map(nats10Values, func(i int) []float32 {
// 		return []float32{float32(i), float32(2 * i)}
// 	})
// 	floats := AppendAll(floatsNested...)
// 	assert.Equal(t, float32(55+55*2), Sum(floats))
// }

// func TestSet(t *testing.T) {
// 	t.Parallel()
// 	intsDuplicated := FlatMap(nats10Values, func(i int) []int {
// 		return Map(nats10Values, func(j int) int { return i + j })
// 	})
// 	intsSet := ToSet(intsDuplicated)
// 	assert.Equal(t, 19, SetSize(intsSet))
// }

// func TestGroupBy(t *testing.T) {
// 	t.Parallel()
// 	intsDuplicated := FlatMap(nats10Values, func(i int) []int {
// 		return Map(nats10Values, func(j int) int { return i + j })
// 	})
// 	intsGroups := GroupBy(intsDuplicated, fun.Identity[int])
// 	assert.Equal(t, 19, len(intsGroups))
// 	for k, as := range intsGroups {
// 		assert.Equal(t, k, as[0])
// 	}
// }

// func TestSliding(t *testing.T) {
// 	t.Parallel()
// 	intWindows := Sliding(nats10Values, 3, 2)
// 	assert.ElementsMatch(t, intWindows[4], []int{9, 10})
// 	intWindows = Sliding(nats10Values, 2, 5)
// 	assert.ElementsMatch(t, intWindows[1], []int{6, 7})
// }

// func TestGrouped(t *testing.T) {
// 	t.Parallel()
// 	intWindows := Grouped(nats10Values, 3)
// 	assert.ElementsMatch(t, intWindows[3], []int{10})
// }

// func TestGroupByMapCount(t *testing.T) {
// 	t.Parallel()
// 	counted := GroupByMapCount(nats10Values, IsEven)
// 	assert.Equal(t, 5, counted[false])
// 	assert.Equal(t, 5, counted[true])
// }
