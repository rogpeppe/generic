package chans

import (
	"sync"

	"github.com/rogpeppe/generic/heap"
)

// Merge returns a channel that receives all the values read from cs.
//
// If less is non-nil, it assumes that the values received from each channel
// are ordered according to the less function - that ordering
// will be maintained in the returned values.
func Merge[T any](cs []<-chan T, less func(T, T) bool) <-chan T {
	if len(cs) == 0 {
		return Closed[T]()
	}
	if len(cs) == 1 {
		return cs[0]
	}
	rc := make(chan T)
	if less != nil {
		go mergeOrdered(cs, less, rc)
	} else {
		go mergeUnordered(cs, rc)
	}
	return rc
}

func mergeUnordered[T any](cs []<-chan T, rc chan<- T) {
	defer close(rc)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		c := c
		go func() {
			defer wg.Done()
			for v := range c {
				rc <- v
			}
		}()
	}
}

type heapEntry[T any] struct {
	x     T
	index int
}

func mergeOrdered[T any](cs []<-chan T, less func(T, T) bool, rc chan<- T) {
	defer close(rc)
	items := heap.New[heapEntry[T]](nil, func(e1, e2 heapEntry[T]) bool {
		return less(e1.x, e2.x)
	}, nil)
	for i, c := range cs {
		if x, ok := <-c; ok {
			items.Push(heapEntry[T]{
				x:     x,
				index: i,
			})
		} else {
			cs[i] = nil
		}
	}
	for items.Len() > 0 {
		item := items.Pop()
		rc <- item.x
		if x, ok := <-cs[item.index]; ok {
			items.Push(heapEntry[T]{
				x:     x,
				index: item.index,
			})
		}
	}
	close(rc)
}

// Closed returns a closed channel with element type T.
func Closed[T any]() <-chan T {
	c := make(chan T)
	close(c)
	return c
}
