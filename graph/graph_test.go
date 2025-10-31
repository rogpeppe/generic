package graph

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"testing"
)

type graphTest struct {
	arcs     [][2]int
	from, to int
	want     [][2]int
}

var graphTests = []graphTest{{
	arcs: [][2]int{
		{0, 1},
		{2, 0},
		{2, 4},
		{2, 5},
		{2, 3},
		{1, 5},
		{2, 5},
	},
	from: 0,
	to:   5,
	want: [][2]int{
		{0, 1},
		{1, 5},
	},
}, {
	arcs: [][2]int{
		{0, 1},
		{0, 2},
		{0, 3},
		{2, 3},
		{3, 4},
		{4, 2},
		{4, 5},
		{7, 0},
	},
	from: 7,
	to:   5,
	want: [][2]int{
		{7, 0},
		{0, 3},
		{3, 4},
		{4, 5},
	},
}}

func TestShortestPath(t *testing.T) {
	for i, test := range graphTests {
		t.Run(fmt.Sprint("test", i), func(t *testing.T) {
			testShortestPath(t, test)
		})
	}
}

func testShortestPath(t *testing.T, test graphTest) {
	g := newGraph(test.arcs)
	// Note: the results are in reverse order.
	path := ShortestPath(g, test.from, test.to)
	var got [][2]int
	for _, e := range path {
		got = append(got, e)
	}
	if !reflect.DeepEqual(got, test.want) {
		t.Fatalf("unexpected result; got %#v want %#v", got, test.want)
	}
}

func newGraph(edges [][2]int) Graph[int, [2]int] {
	var g Simple[int]
	for _, e := range edges {
		g.AddEdge(e[0], e[1])
	}
	return &g
}

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

func TestIntGraph(t *testing.T) {
	g := intGraph{
		0: {1, 2},
		1: {5},
		2: {0, 4, 5},
		4: {},
		5: {},
	}
	fmt.Println(ShortestPath[int, [2]int](g, 0, 4))
}

type NodeConstraint[Edge any] interface {
	cmp.Ordered
	comparable
	Edges() []Edge
}

type EdgeConstraint[Node comparable] interface {
	Nodes() (from, to Node)
}

func ShortestPathOld[Node NodeConstraint[Edge], Edge EdgeConstraint[Node]](from, to Node) []Edge {
	return ShortestPath[Node, Edge](constrainedGraph[Node, Edge]{}, from, to)
}

type constrainedGraph[Node NodeConstraint[Edge], Edge EdgeConstraint[Node]] struct{}

func (g constrainedGraph[Node, Edge]) CmpNode(n0, n1 Node) int {
	return cmp.Compare(n0, n1)
}

func (g constrainedGraph[Node, Edge]) EdgesFrom(n Node) ([]Edge, bool) {
	return n.Edges(), true
}

func (g constrainedGraph[Node, Edge]) Nodes(e Edge) (from, to Node) {
	return e.Nodes()
}

func (g constrainedGraph[Node, Edge]) AllNodes() iter.Seq[Node] {
	panic("unimplemented")
}
