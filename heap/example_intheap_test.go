// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This example demonstrates an integer heap built using the
// Heap type.
package heap_test

import (
	"fmt"

	"github.com/rogpeppe/generic/heap"
)

// This example inserts several ints into an IntHeap, checks the minimum,
// and removes them in order of priority.
func Example_intHeap() {
	h := heap.New([]int{2, 1, 5}, func(a, b int) bool {
		return a < b
	}, nil)
	h.Push(3)
	fmt.Printf("minimum: %d\n", h.Items[0])
	for h.Len() > 0 {
		fmt.Printf("%d ", h.Pop())
	}
	// Output:
	// minimum: 1
	// 1 2 3 5
}
