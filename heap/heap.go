// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package heap provides heap operations on a slice of values.
// A heap is a tree with the property that each node is the
// minimum-valued node in its subtree.
//
// The minimum element in the tree is the root, at index 0.
//
// A heap is a common way to implement a priority queue. To build a priority
// queue, implement the Heap interface with the (negative) priority as the
// ordering for the Less method, so Push adds items while Pop removes the
// highest-priority item from the queue. The Examples include such an
// implementation; the file example_pq_test.go has the complete source.
//
package heap

// New returns a binary heap on the items slice, using less to compare.
// If setIndex is non-nil, it will be called when an item in the heap
// is moved, and passed a pointer to the item that has moved
// and its new index in the slice.
func New[E any](items []E, less func(E, E) bool, setIndex func(e *E, i int)) *Heap[E] {
	h := &Heap[E]{
		Items:    items,
		less:     less,
		setIndex: setIndex,
	}
	h.Init()
	return h
}

// Heap implements a binary heap.
type Heap[E any] struct {
	// Items holds all the items in the heap. The first item is less
	// than all the others.
	Items    []E
	less     func(E, E) bool
	setIndex func(*E, int)
}

// Len returns the number of items in the heap.
func (h *Heap[E]) Len() int {
	return len(h.Items)
}

// Init establishes the heap invariants required by the other routines in this package.
// Init is idempotent with respect to the heap invariants
// and may be called whenever the heap invariants may have been invalidated.
// The complexity is O(n) where n = h.Len().
func (h *Heap[E]) Init() {
	n := len(h.Items)
	for i := n/2 - 1; i >= 0; i-- {
		h.down(i, n)
	}
}

// Push pushes the element x onto the heap.
// The complexity is O(log n) where n = h.Len().
func (h *Heap[E]) Push(x E) {
	h.Items = append(h.Items, x)
	if h.setIndex != nil {
		index := len(h.Items) - 1
		h.setIndex(&h.Items[index], index)
	}
	h.up(len(h.Items) - 1)
}

// Pop removes and returns the minimum element (according to the less function) from the heap.
// The complexity is O(log n) where n = h.Len().
// Pop is equivalent to Remove(h, 0).
func (h *Heap[E]) Pop() E {
	n := len(h.Items) - 1
	h.swap(0, n)
	h.down(0, n)
	return h.pop()
}

// Fix re-establishes the heap ordering after the element at index i has changed its value.
// Changing the value of the element at index i and then calling Fix is equivalent to,
// but less expensive than, calling Remove(h, i) followed by a Push of the new value.
// The complexity is O(log n) where n = h.Len().
func (h *Heap[E]) Fix(i int) {
	if !h.down(i, len(h.Items)) {
		h.up(i)
	}
}

func (h *Heap[E]) swap(i, j int) {
	h.Items[i], h.Items[j] = h.Items[j], h.Items[i]
	if h.setIndex != nil {
		h.setIndex(&h.Items[i], i)
		h.setIndex(&h.Items[j], j)
	}
}

// Remove removes and returns the element at index i from the heap.
// The complexity is O(log n) where n = h.Len().
func (h *Heap[E]) Remove(i int) E {
	n := len(h.Items) - 1
	if n != i {
		h.swap(i, n)
		if !h.down(i, n) {
			h.up(i)
		}
	}
	return h.pop()
}

func (h *Heap[E]) pop() E {
	n := len(h.Items) - 1
	x := h.Items[n]
	h.Items = h.Items[0:n]
	return x
}

func (h *Heap[E]) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !h.less(h.Items[j], h.Items[i]) {
			break
		}
		h.swap(i, j)
		j = i
	}
}

func (h *Heap[E]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.less(h.Items[j2], h.Items[j1]) {
			j = j2 // = 2*i + 2  // right child
		}
		if !h.less(h.Items[j], h.Items[i]) {
			break
		}
		h.swap(i, j)
		i = j
	}
	return i > i0
}
