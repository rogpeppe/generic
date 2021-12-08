// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This example demonstrates a priority queue built using the Heap type.
package heap_test

import (
	"fmt"

	"github.com/rogpeppe/generic/heap"
)

// An Item is something we manage in a priority queue.
type Item struct {
	value    string // The value of the item; arbitrary.
	priority int    // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

func (i *Item) less(j *Item) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return i.priority > j.priority
}

// This example creates a PriorityQueue with some items, adds and manipulates an item,
// and then removes the items in priority order.
func Example_priorityQueue() {
	// Some items and their priorities.
	itemsMap := map[string]int{
		"banana": 3,
		"apple":  2,
		"pear":   4,
	}

	// Create a priority queue, put the items in it, and
	// establish the priority queue (heap) invariants.
	items := make([]*Item, len(itemsMap))
	i := 0
	for value, priority := range itemsMap {
		items[i] = &Item{
			value:    value,
			priority: priority,
			index:    i,
		}
		i++
	}
	pq := heap.New(items, (*Item).less, func(i **Item, index int) {
		(*i).index = index
	})

	// Insert a new item and then modify its priority.
	item := &Item{
		value:    "orange",
		priority: 1,
	}
	pq.Push(item)
	item.priority = 5
	pq.Fix(item.index)

	// Take the items out; they arrive in decreasing priority order.
	for pq.Len() > 0 {
		item := pq.Pop()
		fmt.Printf("%.2d:%s ", item.priority, item.value)
	}
	// Output:
	// 05:orange 04:pear 03:banana 02:apple
}
