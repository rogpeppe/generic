// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package heap

import (
	"math/rand"
	"testing"
)

func newIntHeap(items []int) *Heap[int] {
	return New(items, func(a, b int) bool {
		return a < b
	}, nil)
}

func verifyHeap(t *testing.T, h *Heap[int], i int) {
	t.Helper()
	n := len(h.Items)
	j1 := 2*i + 1
	j2 := 2*i + 2
	if j1 < n {
		if h.Items[j1] < h.Items[i] {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, h.Items[i], j1, h.Items[j1])
			return
		}
		verifyHeap(t, h, j1)
	}
	if j2 < n {
		if h.Items[j2] < h.Items[i] {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, h.Items[i], j1, h.Items[j2])
			return
		}
		verifyHeap(t, h, j2)
	}
}

func TestInit0(t *testing.T) {
	var items []int
	for i := 20; i > 10; i-- {
		items = append(items, 0) // all elements are the same
	}
	h := newIntHeap(items)
	verifyHeap(t, h, 0)

	for i := 1; len(h.Items) > 0; i++ {
		x := h.Pop()
		verifyHeap(t, h, 0)
		if x != 0 {
			t.Errorf("%d.th pop got %d; want %d", i, x, 0)
		}
	}
}

func Test(t *testing.T) {
	var items []int
	for i := 20; i > 10; i-- {
		items = append(items, i)
	}
	h := newIntHeap(items)
	verifyHeap(t, h, 0)

	for i := 10; i > 0; i-- {
		h.Push(i)
		verifyHeap(t, h, 0)
	}

	for i := 1; len(h.Items) > 0; i++ {
		x := h.Pop()
		if i < 20 {
			h.Push(20 + i)
		}
		verifyHeap(t, h, 0)
		if x != i {
			t.Errorf("%d.th pop got %d; want %d", i, x, i)
		}
	}
}

func TestRemove0(t *testing.T) {
	var items []int
	for i := 0; i < 10; i++ {
		items = append(items, i)
	}
	h := newIntHeap(items)

	for len(h.Items) > 0 {
		i := len(h.Items) - 1
		x := h.Remove(i)
		if x != i {
			t.Errorf("Remove(%d) got %d; want %d", i, x, i)
		}
		verifyHeap(t, h, 0)
	}
}

// benchmark          old ns/op     new ns/op     delta
// BenchmarkDup-4     350032        264131        -24.54%
func BenchmarkDup(b *testing.B) {
	const n = 10000
	h := newIntHeap(make([]int, 0, n))
	for i := 0; i < b.N; i++ {
		for j := 0; j < n; j++ {
			h.Push(0) // all elements are the same
		}
		for len(h.Items) > 0 {
			h.Pop()
		}
	}
}

func TestFix(t *testing.T) {
	h := newIntHeap(nil)
	for i := 200; i > 0; i -= 10 {
		h.Push(i)
	}
	verifyHeap(t, h, 0)

	if h.Items[0] != 10 {
		t.Fatalf("Expected head to be 10, was %d", h.Items[0])
	}
	h.Items[0] = 210
	h.Fix(0)
	verifyHeap(t, h, 0)

	for i := 100; i > 0; i-- {
		elem := rand.Intn(len(h.Items))
		if i&1 == 0 {
			h.Items[elem] *= 2
		} else {
			h.Items[elem] /= 2
		}
		h.Fix(elem)
		verifyHeap(t, h, 0)
	}
}
