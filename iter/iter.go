package main
import (
	"bufio"
	"io"
)

type Iter[T any] interface {
	Next() bool
	Item() T
	Err() error
}

func Slice[T any](xs []T) Iter[T] {
	return &sliceIter[T]{
		xs: xs,
		index: -1,
	}
}

type sliceIter[T any] struct {
	xs []T
	index int
}

func (i *sliceIter[T]) Next() bool {
	i.index++
	return i.index < len(i.xs)
}

func (i *sliceIter[T]) Item() T {
	var x T
	if i.index < len(i.xs) {
		x = i.xs[i.index]
	}
	return x
}

func (i *sliceIter[T]) Err() error {
	return nil
}

func Lines(r io.Reader) Iter[string] {
	return linesIter{
		scanner: bufio.NewScanner(r),
	}
}

type linesIter struct {
	scanner *bufio.Scanner
}

func (i linesIter) Next() bool {
	return i.scanner.Scan()
}

func (i linesIter) Err() error {
	return i.scanner.Err()
}

func (i linesIter) Item() string {
	return i.scanner.Text()
}

func Map[S, T any](i Iter[S], f func(S) (T, error)) Iter[T] {
	return &mapIter[S, T]{
		f: f,
		iter: i,
	}
}

type mapIter[S, T any] struct {
	f func(S) (T, error)
	err error
	item T
	iter Iter[S]
}

func (i *mapIter[S, T]) Next() bool {
	if !i.iter.Next() {
		return false
	}
	x, err := i.f(i.iter.Item())
	if err != nil {
		i.err = nil
		return false
	}
	i.item = x
	return true
}

func (i *mapIter[S, T]) Item() T {
	return i.item
}

func (i *mapIter[S, T]) Err() error {
	if err := i.iter.Err(); err != nil {
		return err
	}
	return i.err
}

func Reduce[S, T any](i Iter[T], first S, f func(S, T) (S, error)) (S, error) {
	x := first
	for i.Next() {
		y, err := f(x, i.Item())
		if err != nil {
			return x, err
		}
		x = y
	}
	return x, nil
}

func Select[T any](i Iter[T], f func(T) bool) Iter[T] {
	return selectIter[T]{
		iter: i,
		f: f,
	}
}

type selectIter[T any] struct {
	iter Iter[T]
	f func(T) bool
}

func (i selectIter[T]) Next() bool {
	for {
		if !i.iter.Next() {
			return false
		}
		if i.f(i.iter.Item()) {
			return true
		}
	}
}

func (i selectIter[T]) Item() T {
	return i.iter.Item()
}

func (i selectIter[T]) Err() error {
	return i.iter.Err()
}
