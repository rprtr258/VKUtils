package fun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func concat(a string, b string) string {
	return a + b
}

func TestFun(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", ConstUnit("hello")(Unit1))
	assert.Equal(t, "hello", Identity("hello"))
	concatc := Curry(concat)
	assert.Equal(t, "ab", concatc("a")("b"))
	assert.Equal(t, "ba", Swap(concatc)("a")("b"))
}

func TestPair(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "a", NewPair("a", "b").Left)
	assert.Equal(t, "b", NewPair("a", "b").Right)
}

func TestEither(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "left", Fold(Left[string, string]("left"), Identity[string], Const[string]("other")))
	assert.Equal(t, "other", Fold(Right[string]("right"), Identity[string], Const[string]("other")))
}

func TestToString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1", ToString(1))
}
