// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This code is derived from the code in the v.io/x/lib/toposort package.

package topo

import (
	"cmp"
	"iter"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/rogpeppe/generic/graph"
)

func TestSortDag(t *testing.T) {
	// This is the graph:
	// ,-->B
	// |
	// A-->C---->D
	// |    \
	// |     `-->E--.
	// `-------------`-->F
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "F")
	sorted, cycles := TopoSort(g)
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
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("A", "C")
	g.AddEdge("C", "D")
	sorted, cycles := TopoSort(g)
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
	g := new(graph.Simple[string])
	g.AddEdge("A", "A")
	sorted, cycles := TopoSort(g)
	oc := makeOrderChecker(t, sorted)
	oc.expectTotalOrder("A")
	expectCycles(t, cycles, [][]string{{"A", "A"}})
}

func TestSortCycle(t *testing.T) {
	// This is the graph:
	// ,-->B-->C
	// |       |
	// A<------'
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")
	g.AddEdge("C", "A")
	sorted, cycles := TopoSort(g)
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
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("D", "C") // creates the cycle
	g.AddEdge("E", "F")
	sorted, cycles := TopoSort(g)
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
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "F")
	g.AddEdge("F", "C") // creates the cycle
	sorted, cycles := TopoSort(g)
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
	g := new(graph.Simple[string])
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")
	g.AddEdge("A", "F")
	g.AddEdge("C", "D")
	g.AddEdge("C", "E")
	g.AddEdge("E", "A") // creates a cycle
	g.AddEdge("E", "F")
	g.AddEdge("F", "C") // creates a cycle
	sorted, cycles := TopoSort(g)
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

///// gonum tests

type interval struct{ start, end int }

var tarjanTests = []struct {
	g intGraph

	ambiguousOrder []interval
	want           [][]int

	sortedLength      int
	unorderableLength int
	sortable          bool
}{
	{
		g: intGraph{
			0: {1},
			1: {2, 7},
			2: {3, 6},
			3: {4},
			4: {2, 5},
			6: {3, 5},
			7: {0, 6},
		},

		want: [][]int{
			{5},
			{2, 3, 4, 6},
			{0, 1, 7},
		},

		sortedLength:      1,
		unorderableLength: 2,
		sortable:          false,
	},
	{
		g: intGraph{
			0: {1, 2, 3},
			1: {2},
			2: {3},
			3: {1},
		},

		want: [][]int{
			{1, 2, 3},
			{0},
		},

		sortedLength:      1,
		unorderableLength: 1,
		sortable:          false,
	},
	{
		g: intGraph{
			0: {1},
			1: {0, 2},
			2: {1},
		},

		want: [][]int{
			{0, 1, 2},
		},

		sortedLength:      0,
		unorderableLength: 1,
		sortable:          false,
	},
	{
		g: intGraph{
			0: {1},
			1: {2, 3},
			2: {4, 5},
			3: {4, 5},
			4: {6},
			5: nil,
			6: nil,
		},

		// Node pairs (2, 3) and (4, 5) are not
		// relatively orderable within each pair.
		ambiguousOrder: []interval{
			{0, 3}, // This includes node 6 since it only needs to be before 4 in topo sort.
			{3, 5},
		},
		want: [][]int{
			{6}, {5}, {4}, {3}, {2}, {1}, {0},
		},

		sortedLength: 7,
		sortable:     true,
	},
	{
		g: intGraph{
			0: {1},
			1: {2, 3, 4},
			2: {0, 3},
			3: {4},
			4: {3},
		},

		// SCCs are not relatively ordable.
		ambiguousOrder: []interval{
			{0, 2},
		},
		want: [][]int{
			{0, 1, 2},
			{3, 4},
		},

		sortedLength:      0,
		unorderableLength: 2,
		sortable:          false,
	},
}

func TestSort(t *testing.T) {
	for i, test := range tarjanTests {
		sorted, err := Sort(test.g)
		gotSortedLen := len(sorted)
		if gotSortedLen != test.sortedLength {
			t.Errorf("unexpected number of sortable nodes for test %d: got:%d want:%d", i, gotSortedLen, test.sortedLength)
		}
		if err == nil != test.sortable {
			t.Errorf("unexpected sortability for test %d: got error: %v want: nil-error=%t", i, err, test.sortable)
		}
		if err != nil && len(err.(Unorderable[int])) != test.unorderableLength {
			t.Errorf("unexpected number of unorderable nodes for test %d: got:%d want:%d", i, len(err.(Unorderable[int])), test.unorderableLength)
		}
	}
}

func TestTarjanSCC(t *testing.T) {
	for i, test := range tarjanTests {
		gotSCCs := TarjanSCC(test.g)
		// tarjan.strongconnect does range iteration over maps,
		// so sort SCC members to ensure consistent ordering.
		for _, scc := range gotSCCs {
			slices.Sort(scc)
		}
		for _, iv := range test.ambiguousOrder {
			slices.SortFunc(test.want[iv.start:iv.end], slices.Compare)
			slices.SortFunc(gotSCCs[iv.start:iv.end], slices.Compare)
		}
		if !reflect.DeepEqual(gotSCCs, test.want) {
			t.Errorf("unexpected Tarjan scc result for %d:\n\tgot:%v\n\twant:%v", i, gotSCCs, test.want)
		}
	}
}

//
//var stabilizedSortTests = []struct {
//	g []intset
//
//	want []int
//	err  error
//}{
//	{
//		g: intGraph{
//			0: linksTo(1),
//			1: linksTo(2, 7),
//			2: linksTo(3, 6),
//			3: linksTo(4),
//			4: linksTo(2, 5),
//			6: linksTo(3, 5),
//			7: linksTo(0, 6),
//		},
//
//		want: []graph.Node{nil, nil, simple.Node(5)},
//		err: Unorderable[int]{
//			{simple.Node(0), simple.Node(1), simple.Node(7)},
//			{simple.Node(2), simple.Node(3), simple.Node(4), simple.Node(6)},
//		},
//	},
//	{
//		g: []intset{
//			0: linksTo(1, 2, 3),
//			1: linksTo(2),
//			2: linksTo(3),
//			3: linksTo(1),
//		},
//
//		want: []graph.Node{simple.Node(0), nil},
//		err: Unorderable{
//			{simple.Node(1), simple.Node(2), simple.Node(3)},
//		},
//	},
//	{
//		g: []intset{
//			0: linksTo(1),
//			1: linksTo(0, 2),
//			2: linksTo(1),
//		},
//
//		want: []graph.Node{nil},
//		err: Unorderable{
//			{simple.Node(0), simple.Node(1), simple.Node(2)},
//		},
//	},
//	{
//		g: []intset{
//			0: linksTo(1),
//			1: linksTo(2, 3),
//			2: linksTo(4, 5),
//			3: linksTo(4, 5),
//			4: linksTo(6),
//			5: nil,
//			6: nil,
//		},
//
//		want: []graph.Node{simple.Node(0), simple.Node(1), simple.Node(2), simple.Node(3), simple.Node(4), simple.Node(5), simple.Node(6)},
//		err:  nil,
//	},
//	{
//		g: []intset{
//			0: linksTo(1),
//			1: linksTo(2, 3, 4),
//			2: linksTo(0, 3),
//			3: linksTo(4),
//			4: linksTo(3),
//		},
//
//		want: []graph.Node{nil, nil},
//		err: Unorderable{
//			{simple.Node(0), simple.Node(1), simple.Node(2)},
//			{simple.Node(3), simple.Node(4)},
//		},
//	},
//	{
//		g: []intset{
//			0: linksTo(1, 2, 3, 4, 5, 6),
//			1: linksTo(7),
//			2: linksTo(7),
//			3: linksTo(7),
//			4: linksTo(7),
//			5: linksTo(7),
//			6: linksTo(7),
//			7: nil,
//		},
//
//		want: []graph.Node{simple.Node(0), simple.Node(1), simple.Node(2), simple.Node(3), simple.Node(4), simple.Node(5), simple.Node(6), simple.Node(7)},
//		err:  nil,
//	},
//}
//
//func TestSortStabilized(t *testing.T) {
//	for i, test := range stabilizedSortTests {
//		var g Simple[int]
//		for u, e := range test.g {
//			if len(e) == 0 {
//				g.AddNode(u)
//			} else {
//				for v := range e {
//					g.AddEdge(u, v)
//				}
//			}
//		}
//		got, err := SortStabilized(g, nil)
//		if !reflect.DeepEqual(got, test.want) {
//			t.Errorf("unexpected sort result for test %d: got:%d want:%d", i, got, test.want)
//		}
//		if !reflect.DeepEqual(err, test.err) {
//			t.Errorf("unexpected sort error for test %d: got:%v want:%v", i, err, test.want)
//		}
//	}
//}
//
//// intset is an integer set.
//type intset map[int]struct{}
//
//func linksTo(i ...int64) []int {
//	if len(i) == 0 {
//		return nil
//	}
//	s := make(intset)
//	for _, v := range i {
//		s[v] = struct{}{}
//	}
//	return s
//}

type intGraph map[int][]int

func (g intGraph) CmpNode(n0, n1 int) int {
	return cmp.Compare(n0, n1)
}

func (g intGraph) AllNodes() iter.Seq[int] {
	return maps.Keys(g)
}

func (g intGraph) EdgesFrom(n int) ([][2]int, bool) {
	to, ok := g[n]
	if !ok {
		return nil, false
	}
	edges := make([][2]int, len(to))
	for i, e := range to {
		edges[i] = [2]int{n, e}
	}
	return edges, true
}

func (g intGraph) Nodes(e [2]int) (from, to int) {
	return e[0], e[1]
}
