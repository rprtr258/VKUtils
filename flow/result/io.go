package result

import (
	"github.com/pkg/errors"
	"github.com/rprtr258/vk-utils/flow/fun"
)

// Result represents a calculation that will yield a value of type A once executed.
// The calculation might as well fail.
// It is designed to not panic ever.
type Result[A any] fun.Either[A, error]

func myPanic[A, B any](a A) B {
	panic(a)
}

// Unwrap returns value if present, panics otherwise.
func (r Result[A]) Unwrap() A {
	return Fold(r, fun.Identity[A], myPanic[error, A])
}

// Unwrap returns error if present, panics otherwise.
func (r Result[A]) UnwrapErr() error {
	return Fold(r, myPanic[A, error], fun.Identity[error])
}

// FromGoResult constructs Result from Go result/error pair.
func FromGoResult[A any](a A, err error) Result[A] {
	if err != nil {
		return Result[A](fun.Right[A](err))
	}
	return Result[A](fun.Left[A, error](a))
}

// Eval constructs Result from a function that might fail.
// If there is panic in the function, it's recovered from and represented as an error.
func Eval[A any](f func() (A, error)) Result[A] {
	var (
		a   A
		err error
	)
	defer RecoverToErrorVar("Eval.unsafeRun", &err)
	a, err = f()
	return FromGoResult(a, err)
}

// TryRecover tries to recover result from error using a function that might fail.
func TryRecover[A any](ma Result[A], f func(error) Result[A]) Result[A] {
	return Fold(ma, Success[A], f)
}

// Map converts the IO[A] result using the provided function that cannot fail.
func Map[A, B any](mx Result[A], f func(A) B) Result[B] {
	return fun.Fold(fun.Either[A, error](mx), fun.Compose(f, Success[B]), Fail[B])
}

// Fail constructs an IO[A] that fails with the given error.
func Fail[A any](err error) Result[A] {
	return Result[A](fun.Right[A](err))
}

// FlatMap converts the result of IO[A] using a function that itself returns an IO[B].
// It'll fail if any of IO[A] or IO[B] fail.
func FlatMap[A, B any](mx Result[A], f func(A) Result[B]) Result[B] {
	return fun.Fold(fun.Either[A, error](mx), f, Fail[B])
}

// Success constructs an IO[A] from a constant value.
func Success[A any](a A) Result[A] {
	return Result[A](fun.Left[A, error](a))
}

// Fold constructs value by either success or fail paths.
func Fold[A, B any](mx Result[A], fSuccess func(A) B, fFail func(error) B) B {
	return fun.Fold(fun.Either[A, error](mx), fSuccess, fFail)
}

// FoldConsume consumes value or error and executes according callback.
func FoldConsume[A any](mx Result[A], fSuccess func(A), fFail func(error)) {
	fun.FoldConsume(fun.Either[A, error](mx), fSuccess, fFail)
}

var ErrorNPE = errors.New("nil pointer")

// Dereference retrieves the value by pointer. Fails if pointer is nil.
func Dereference[A any](ptra *A) Result[A] {
	if ptra == nil {
		return Fail[A](ErrorNPE)
	} else {
		return Success(*ptra)
	}
}

// WrapErrf wraps an error with additional context information
func WrapErrf[A any](io Result[A], format string, args ...interface{}) Result[A] {
	return TryRecover(io, func(err error) Result[A] {
		return Fail[A](errors.Wrapf(err, format, args...))
	})
}
