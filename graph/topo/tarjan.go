package topo

import (
	"fmt"
	"slices"

	"github.com/rogpeppe/generic/graph"
)

// Sort performs a topological sort of the directed graph g returning the 'from' to 'to'
// sort order. If a topological ordering is not possible, an Unorderable error is returned
// listing cyclic components in g with each cyclic component's members sorted by ID. When
// an Unorderable error is returned, each cyclic component's topological position within
// the sorted nodes is marked with a nil graph.Node.
func Sort[Node comparable, Edge any](g graph.EnumerableGraph[Node, Edge]) (sorted []Node, err error) {
	sccs := TarjanSCC(g)
	return sortedFrom(sccs, g.CmpNode)
}

// SortStabilized performs a topological sort of the directed graph g returning the 'from'
// to 'to' sort order, or the order defined by the in place order sort function where there
// is no unambiguous topological ordering. If a topological ordering is not possible, an
// Unorderable error is returned listing cyclic components in g with each cyclic component's
// members sorted by the provided order function. If cmp is nil, nodes are ordered
// according to g.CmpNode.
// by node ID. When an Unorderable error is returned, each cyclic component's topological
// position within the sorted nodes is marked with a nil Node.
func SortStabilized[Node comparable, Edge any](g graph.EnumerableGraph[Node, Edge], cmp func(n0, n1 Node) int) (sorted []Node, err error) {
	if cmp == nil {
		cmp = g.CmpNode
	}
	sccs := tarjanSCCstabilized(g, cmp)
	return sortedFrom(sccs, cmp)
}

func sortedFrom[Node comparable](sccs [][]Node, cmp func(n0, n1 Node) int) ([]Node, error) {
	sorted := make([]Node, 0, len(sccs))
	var sc Unorderable[Node]
	for _, s := range sccs {
		if len(s) != 1 {
			slices.SortFunc(s, cmp)
			sc = append(sc, s)
			// TODO the original code marked the position of the cyclic
			// component with a nil node, but we can't do that,
			// and the zero node might be valid.
			// For now just append the zero Node, but perhaps there
			// should be provision for a sentinel invalid Node.
			//sorted = append(sorted, *new(Node))
			continue
		}
		sorted = append(sorted, s[0])
	}
	var err error
	if sc != nil {
		slices.Reverse(sc)
		err = sc
	}
	slices.Reverse(sorted)
	return sorted, err
}

// TarjanSCC returns the strongly connected components of the graph g using Tarjan's algorithm.
//
// A strongly connected component of a graph is a set of vertices where it's possible to reach any
// vertex in the set from any other (meaning there's a cycle between them.)
//
// Generally speaking, a directed graph where the number of strongly connected components is equal
// to the number of nodes is acyclic, unless you count reflexive edges as a cycle (which requires
// only a little extra testing.)
func TarjanSCC[Node comparable, Edge any](g graph.EnumerableGraph[Node, Edge]) [][]Node {
	return tarjanSCCstabilized(g, nil)
}

func tarjanSCCstabilized[Node comparable, Edge any](g graph.EnumerableGraph[Node, Edge], cmp func(n0, n1 Node) int) [][]Node {
	nodes := slices.Collect(g.AllNodes())
	var succ func(id Node) []Node
	if cmp == nil {
		succ = func(n Node) []Node {
			return slices.Collect(graph.NodesFrom(g, n))
		}
	} else {
		slices.SortFunc(nodes, cmp)
		slices.Reverse(nodes)

		succ = func(n Node) []Node {
			to := slices.SortedFunc(graph.NodesFrom(g, n), cmp)
			slices.Reverse(to)
			return to
		}
	}

	t := tarjan[Node, Edge]{
		succ: succ,

		indexTable: make(map[Node]int, len(nodes)),
		lowLink:    make(map[Node]int, len(nodes)),
		onStack:    make(map[Node]bool),
	}
	for _, v := range nodes {
		if t.indexTable[v] == 0 {
			t.strongconnect(v)
		}
	}
	return t.sccs
}

// tarjan implements Tarjan's strongly connected component finding
// algorithm. The implementation is from the pseudocode at
//
// http://en.wikipedia.org/wiki/Tarjan%27s_strongly_connected_components_algorithm?oldid=642744644
type tarjan[Node comparable, Edge any] struct {
	succ func(Node) []Node

	index      int
	indexTable map[Node]int
	lowLink    map[Node]int
	onStack    map[Node]bool

	stack []Node

	sccs [][]Node
}

// strongconnect is the strongconnect function described in the
// wikipedia article.
func (t *tarjan[Node, Edge]) strongconnect(v Node) {

	// Set the depth index for v to the smallest unused index.
	t.index++
	t.indexTable[v] = t.index
	t.lowLink[v] = t.index
	t.stack = append(t.stack, v)
	t.onStack[v] = true

	// Consider successors of v.
	for _, w := range t.succ(v) {
		if t.indexTable[w] == 0 {
			// Successor w has not yet been visited; recur on it.
			t.strongconnect(w)
			t.lowLink[v] = min(t.lowLink[v], t.lowLink[w])
		} else if t.onStack[w] {
			// Successor w is in stack s and hence in the current SCC.
			t.lowLink[v] = min(t.lowLink[v], t.indexTable[w])
		}
	}

	// If v is a root node, pop the stack and generate an SCC.
	if t.lowLink[v] == t.indexTable[v] {
		// Start a new strongly connected component.
		var (
			scc []Node
			w   Node
		)
		for {
			w, t.stack = t.stack[len(t.stack)-1], t.stack[:len(t.stack)-1]
			delete(t.onStack, w)
			// Add w to current strongly connected component.
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		// Output the current strongly connected component.
		t.sccs = append(t.sccs, scc)
	}
}

// Unorderable is an error containing sets of unorderable graph.Nodes.
type Unorderable[Node comparable] [][]Node

// Error satisfies the error interface.
func (e Unorderable[Node]) Error() string {
	const maxNodes = 10
	var n int
	for _, c := range e {
		n += len(c)
	}
	if n > maxNodes {
		// Don't return errors that are too long.
		return fmt.Sprintf("topo: no topological ordering: %d nodes in %d cyclic components", n, len(e))
	}
	return fmt.Sprintf("topo: no topological ordering: cyclic components: %v", [][]Node(e))
}
