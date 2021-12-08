// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This code is derived from the code in the v.io/x/lib/toposort package.

package graph

import (
	"reflect"
	"testing"
)

func TestSortDag(t *testing.T) {
	// This is the graph:
	// ,-->B
	// |
	// A-->C---->D
	// |    \
	// |     `-->E--.
	// `-------------`-->F
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "F")
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectOrder("B", "A")
	oc.expectOrder("C", "A")
	oc.expectOrder("D", "A")
	oc.expectOrder("E", "A")
	oc.expectOrder("F", "A")
	oc.expectOrder("D", "C")
	oc.expectOrder("E", "C")
	oc.expectOrder("F", "C")
	oc.expectOrder("F", "E")
	oc.expectTotalOrder("B", "D", "F", "E", "C", "A")
	expectCycles(t, cycles, [][]string{})
}

func TestSortDagJoin(t *testing.T) {
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("A", "C")
	g.AddEdge("C", "D")
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectOrder("B", "A")
	oc.expectOrder("C", "A")
	oc.expectOrder("C", "B")
	oc.expectOrder("D", "C")
	oc.expectOrder("D", "B")
	oc.expectOrder("D", "A")
	oc.expectTotalOrder("D", "C", "B", "A")
	expectCycles(t, cycles, [][]string{})
}

func TestSortSelfCycle(t *testing.T) {
	// This is the graph:
	// ,---.
	// |   |
	// A<--'
	g := new(Simple[string])
	g.AddEdge("A", "A")
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectTotalOrder("A")
	expectCycles(t, cycles, [][]string{{"A", "A"}})
}

func TestSortCycle(t *testing.T) {
	// This is the graph:
	// ,-->B-->C
	// |       |
	// A<------'
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", "A")
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectTotalOrder("C", "B", "A")
	expectCycles(t, cycles, [][]string{{"A", "C", "B", "A"}})
}

func TestSortContainsCycle1(t *testing.T) {
	// This is the graph:
	// ,-->B
	// |   ,-----.
	// |   v     |
	// A-->C---->D
	// |    \
	// |     `-->E--.
	// `-------------`-->F
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("D", "C") // creates the cycle
	g.AddEdge("E", "F")
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectOrder("B", "A")
	oc.expectOrder("C", "A")
	oc.expectOrder("D", "A")
	oc.expectOrder("E", "A")
	oc.expectOrder("F", "A")
	// The difference with the dag is C, D may be in either order.
	oc.expectOrder("E", "C")
	oc.expectOrder("F", "C")
	oc.expectOrder("F", "E")
	oc.expectTotalOrder("B", "D", "F", "E", "C", "A")
	expectCycles(t, cycles, [][]string{{"C", "D", "C"}})
}

func TestSortContainsCycle2(t *testing.T) {
	// This is the graph:
	// ,-->B
	// |   ,-------------.
	// |   v             |
	// A-->C---->D       |
	// |    \            |
	// |     `-->E--.    |
	// `-------------`-->F
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "F")
	g.AddEdge("F", "C") // creates the cycle
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectOrder("B", "A")
	oc.expectOrder("C", "A")
	oc.expectOrder("D", "A")
	oc.expectOrder("E", "A")
	oc.expectOrder("F", "A")
	oc.expectOrder("D", "C")
	// The difference with the dag is C, E, F may be in any order.
	oc.expectTotalOrder("B", "D", "F", "E", "C", "A")
	expectCycles(t, cycles, [][]string{{"C", "F", "E", "C"}})
}

func TestSortMultiCycles(t *testing.T) {
	// This is the graph:
	//    ,-->B
	//    |   ,------------.
	//    |   v            |
	// .--A-->C---->D      |
	// |  ^    \           |
	// |  |     `-->E--.   |
	// |  |         |  |   |
	// |  `---------'  |   |
	// `---------------`-->F
	g := new(Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "A") // creates a cycle
	g.AddEdge("E", "F")
	g.AddEdge("F", "C") // creates a cycle
	sorted, cycles := TopoSort(g.Graph())
	oc := makeOrderChecker(t, sorted)
	oc.expectOrder("B", "A")
	oc.expectOrder("D", "A")
	oc.expectOrder("F", "A")
	oc.expectOrder("D", "C")
	oc.expectTotalOrder("B", "D", "F", "E", "C", "A")
	expectCycles(t, cycles, [][]string{{"A", "E", "C", "A"}, {"C", "F", "E", "C"}})
}

type orderChecker struct {
	t        *testing.T
	original []string
	orderMap map[string]int
}

func makeOrderChecker(t *testing.T, slice []string) orderChecker {
	result := orderChecker{t, slice, make(map[string]int)}
	for ix, val := range result.original {
		result.orderMap[val] = ix
	}
	return result
}

func (oc *orderChecker) findValue(val string) int {
	if index, ok := oc.orderMap[val]; ok {
		return index
	}
	oc.t.Errorf("Couldn't find val %v in slice %v", val, oc.original)
	return -1
}

func (oc *orderChecker) expectOrder(before, after string) {
	if oc.findValue(before) >= oc.findValue(after) {
		oc.t.Errorf("Expected %v before %v, slice %v", before, after, oc.original)
	}
}

// Since sort is deterministic we can expect a particular total order, in
// addition to the partial order checks.
func (oc *orderChecker) expectTotalOrder(expect ...string) {
	oc.t.Helper()
	if !reflect.DeepEqual(oc.original, expect) {
		oc.t.Errorf("Expected order %v, actual %v", expect, oc.original)
	}
}

func expectCycles(t *testing.T, actual [][]string, expect [][]string) {
	if len(actual) == 0 {
		actual = nil
	}
	if len(expect) == 0 {
		expect = nil
	}
	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("Expected cycles %v, actual %v", expect, actual)
	}
}
