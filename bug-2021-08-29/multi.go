package main

func main() {
	Fooer2[byte]()
}

type Fooer[T any] interface {
	Foo(p T)
}

type fooer1[T any] struct{}

func (fooer1[T]) Foo(T) {}

type fooer2[T any] struct {
	r []Fooer[T]
}

func (mr fooer2[T]) Foo(p T) {
	mr.r[0] = fooer1[T]{}
	return
}

func Fooer2[T any]() Fooer[T] {
	return fooer2[T]{}
}
