package graph

import "iter"

type Graph[Node comparable, Edge any] interface {
	// TODO return iterator?
	// TODO it's useful to be able to tell if a given node
	// is actually part of the graph or not. Could
	// say that returning nil slice signifies non-presence,
	// or (probably better) return ([]Edge, bool).
	EdgesFrom(Node) ([]Edge, bool)
	Nodes(Edge) (from, to Node)
	CmpNode(n0, n1 Node) int
}

// SimpleGraph is a special case of [Graph]
// that only allows a single edge between any two
// nodes. An edge may not connect a node to itself.
type SimpleGraph[Node comparable, Edge any] interface {
	Graph[Node, Edge]
	Edge(Node) (Edge, bool)
}

type EnumerableGraph[Node comparable, Edge any] interface {
	Graph[Node, Edge]
	AllNodes() iter.Seq[Node]
}

type EnumerableSimpleGraph[Node comparable, Edge any] interface {
	SimpleGraph[Node, Edge]
	AllNodes() iter.Seq[Node]
}

type Weighted[Node comparable, Edge any] interface {
	Graph[Node, Edge]

	// EdgeWeight returns the weight of the given edge.
	EdgeWeight(Edge) float64
}

func NodesFrom[Node comparable, Edge any](g Graph[Node, Edge], n Node) iter.Seq[Node] {
	return func(yield func(Node) bool) {
		edges, _ := g.EdgesFrom(n)
		for _, e := range edges {
			if _, to := g.Nodes(e); !yield(to) {
				break
			}
		}
	}
}

func NodeInGraph[Node comparable, Edge any](g Graph[Node, Edge], n Node) bool {
	_, ok := g.EdgesFrom(n)
	return ok
}
