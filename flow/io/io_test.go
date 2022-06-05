package io

// import (
// 	"errors"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestIO(t *testing.T) {
// 	io10 := Lift(10)
// 	io20 := Map(io10, func(i int) int { return i * 2 })
// 	io30 := FlatMap(io10, func(i int) IO[int] {
// 		return MapErr(io20, func(j int) (int, error) {
// 			return i + j, nil
// 		})
// 	})
// 	res, err := UnsafeRunSync(io30)
// 	assert.Equal(t, err, nil)
// 	assert.Equal(t, res, 30)
// }

// func TestErr(t *testing.T) {
// 	var ptr *string = nil
// 	ptrio := Lift(ptr)
// 	uptr := FlatMap(ptrio, Unptr[string])
// 	_, err := UnsafeRunSync(uptr)
// 	assert.Equal(t, ErrorNPE, err)
// 	wrappedUptr := Wrapf(uptr, "my message %d", 10)
// 	_, err = UnsafeRunSync(wrappedUptr)
// 	assert.Equal(t, "my message 10: nil pointer", err.Error())
// }

// func TestFinally(t *testing.T) {
// 	errorMessage := "on purpose failure"
// 	failure := Fail[string](errors.New(errorMessage))
// 	finalizerExecuted := false
// 	fin := Finally(failure, FromPureEffect(func() { finalizerExecuted = true }))
// 	_, err := UnsafeRunSync(fin)
// 	assert.Error(t, err, errorMessage)
// 	assert.True(t, finalizerExecuted)
// }
