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
