package graph

import (
	"fmt"
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
		{1, 5},
		{0, 1},
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
		{4, 5},
		{3, 4},
		{0, 3},
		{7, 0},
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
	path := ShortestPath(Graph[int, int](g), test.from, test.to)
	var got [][2]int
	for _, e := range path {
		got = append(got, g.edges[e])
	}
	if !reflect.DeepEqual(got, test.want) {
		t.Fatalf("unexpected result; got %#v want %#v", got, test.want)
	}
}

func newGraph(edges [][2]int) *graph {
	nodes := make(map[int][]int)
	for i, e := range edges {
		nodes[e[0]] = append(nodes[e[0]], i)
	}
	return &graph{
		edges: edges,
		nodes: nodes,
	}
}

type graph struct {
	edges [][2]int
	nodes map[int][]int
}

func (g *graph) Edges(n int) []int {
	return g.nodes[n]
}

func (g *graph) Nodes(e int) (from, to int) {
	if e < 0 || e >= len(g.edges) {
		panic(fmt.Errorf("unknown edge id %d", e))
	}
	arc := g.edges[e]
	return arc[0], arc[1]
}
