package resource

// import (
// 	"errors"
// 	"testing"
// 	"time"

// 	"github.com/rprtr258/goflow/fun"
// 	"github.com/rprtr258/goflow/io"
// 	"github.com/stretchr/testify/assert"
// )

// func TestResource(t *testing.T) {
// 	res := NewResource(
// 		io.Lift("resource"),
// 		func(s string) io.IO[fun.Unit] {
// 			assert.Equal(t, "resource", s)
// 			return io.IOUnit1
// 		},
// 	)
// 	resMapped := Map(res, func(s string) int {
// 		return len(s)
// 	})
// 	io9 := Use(resMapped, func(i int) io.IO[int] {
// 		return io.Lift(i + 1)
// 	})
// 	res9, err := io.UnsafeRunSync(io9)
// 	assert.Equal(t, err, nil)
// 	assert.Equal(t, res9, 9)
// }
// func TestResourceFail(t *testing.T) {
// 	released := false
// 	res := NewResource(
// 		io.Lift("resource"),
// 		func(s string) io.IO[fun.Unit] {
// 			assert.Equal(t, "resource", s)
// 			released = true
// 			return io.IOUnit1
// 		},
// 	)

// 	failed := Use(res, func(s string) io.IO[int] {
// 		return io.Fail[int](errors.New("some error"))
// 	})
// 	_, err := io.UnsafeRunSync(failed)
// 	assert.NotEqual(t, err, nil)
// 	assert.True(t, released)
// }

// func TestChannelResource(t *testing.T) {
// 	stringChannel := UnbufferedChannel[string]()
// 	helloIO := Use(stringChannel, func(ch chan string) io.IO[string] {
// 		notify := io.NotifyToChannel(100*time.Millisecond, "hello", ch)
// 		return io.AndThen(notify, io.FromChannel(ch))
// 	})
// 	hello, err := io.UnsafeRunSync(helloIO)
// 	assert.NoError(t, err)
// 	assert.Equal(t, "hello", hello)
// }

// func TestResourceInResource(t *testing.T) {
// 	res1 := NewResource(
// 		io.Lift("resource1"),
// 		func(s string) io.IO[fun.Unit] {
// 			assert.Equal(t, "resource1", s)
// 			return io.IOUnit1
// 		},
// 	)
// 	res2 := FlatMap(res1, func(s string) Resource[fun.Pair[string, string]] {

// 		return NewResource(
// 			io.Lift(fun.NewPair(s, "resource2")),
// 			func(p fun.Pair[string, string]) io.IO[fun.Unit] {
// 				assert.Equal(t, "resource2", p.Right)
// 				return io.IOUnit1
// 			},
// 		)
// 	})
// 	io18 := Use(res2, func(p fun.Pair[string, string]) io.IO[int] {
// 		return io.Lift(len(p.Left) + len(p.Right))
// 	})
// 	res18, err := io.UnsafeRunSync(io18)
// 	assert.Equal(t, err, nil)
// 	assert.Equal(t, res18, 18)
// }
