package fun

// Either is either A value or B value.
type Either[A, B any] struct {
	left  *A
	right *B
}

// Fold pattern matches Either with two given pattern match handlers
func Fold[A, B, C any](x Either[A, B], fLeft func(A) C, fRight func(B) C) C {
	switch {
	case x.left != nil:
		return fLeft(*x.left)
	default:
		return fRight(*x.right)
	}
}

// Left constructs Either that is left.
func Left[A, B any](a A) Either[A, B] {
	return Either[A, B]{&a, nil}
}

// Right constructs Either that is right.
func Right[A, B any](b B) Either[A, B] {
	return Either[A, B]{nil, &b}
}

// IsLeft checks whether the provided Either is left or not.
func IsLeft[A, B any](x Either[A, B]) bool {
	return Fold(x, Const[A](true), Const[B](false))
}

// IsRight checks whether the provided Either is right or not.
func IsRight[A, B any](x Either[A, B]) bool {
	return Fold(x, Const[A](false), Const[B](true))
}

type Option[A any] Either[A, Unit]

func None[A any]() Option[A] {
	return Option[A](Right[A](Unit1))
}

func Some[A any](a A) Option[A] {
	return Option[A](Left[A, Unit](a))
}

func (x *Option[A]) IsNone() bool {
	return IsRight(Either[A, Unit](*x))
}

func (x *Option[A]) IsSome() bool {
	return IsLeft(Either[A, Unit](*x))
}

func (x Option[A]) Unwrap() A {
	return *x.left
}

func FoldOption[A, B any](x Option[A], fLeft func(A) B, fRight func() B) B {
	return Fold(
		Either[A, Unit](x),
		fLeft,
		func(_ Unit) B { return fRight() },
	)
}

func Map[A, B any](x Option[A], f func(A) B) Option[B] {
	return FoldOption(x, Compose(f, Some[B]), None[B])
}

// TODO: is *A faster?
// TODO: is A*bool faster?
// type Option[A any] struct{ x *A }

// func None[A any]() Option[A] {
// 	return Option[A]{nil}
// }

// func Some[A any](a A) Option[A] {
// 	return Option[A]{&a}
// }

// func (x *Option[A]) IsNone() bool {
// 	return x.x == nil
// }

// func (x *Option[A]) IsSome() bool {
// 	return x.x != nil
// }

// func (x *Option[A]) Unwrap() A {
// 	return *x.x
// }

// func FoldOption[A, B any](x Option[A], fLeft func(A) B, fRight func() B) B {
// 	if x.x == nil {
// 		return fRight()
// 	}
// 	return fLeft(*x.x)
// }
