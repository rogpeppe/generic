package graph

import (
	"cmp"
	"iter"
	"slices"
)

// Simple implements Graph for a concrete set of comparable nodes.
type Simple[Node cmp.Ordered] struct {
	nodes    map[Node][][2]Node
	allNodes []Node
}

func (g *Simple[Node]) CmpNode(n0, n1 Node) int {
	return cmp.Compare(n0, n1)
}

// AddNode adds a node. Typically this is only used to add
// nodes with no incoming or outgoing edges.
func (g *Simple[Node]) AddNode(n Node) {
	g.addNode(n)
}

// AddEdge adds nodes from and to, and adds an edge from -> to.
// You don't need to call AddNode first; the nodes will be implicitly added if they don't
// already exist.  The direction means that from depends on to; i.e. to will
// appear before from in the sorted output.  Cycles are allowed.
func (g *Simple[Node]) AddEdge(from, to Node) {
	g.addNode(from, [2]Node{from, to})
	g.addNode(to)
}

func (g *Simple[Node]) addNode(n Node, edges ...[2]Node) {
	if g.nodes == nil {
		g.nodes = make(map[Node][][2]Node)
	}
	n0 := len(g.nodes)
	g.nodes[n] = append(g.nodes[n], edges...)
	if len(g.nodes) > n0 {
		g.allNodes = append(g.allNodes, n)
	}
}

// AllNodes implements Graph.AllNodes.
func (g *Simple[Node]) AllNodes() iter.Seq[Node] {
	return slices.Values(g.allNodes)
}

// AllNodes implements Graph.Edges.
// Note: the caller should not mutate the returned slice.
func (g *Simple[Node]) EdgesFrom(n Node) ([][2]Node, bool) {
	edges, ok := g.nodes[n]
	return edges, ok
}

// AllNodes implements Graph.Nodes.
func (g *Simple[Node]) Nodes(e [2]Node) (from, to Node) {
	return e[0], e[1]
}
