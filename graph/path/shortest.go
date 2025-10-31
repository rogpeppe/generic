// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package path

import (
	"math"
	"slices"
)

// Shortest is a shortest-path tree created by the BellmanFordFrom, DijkstraFrom
// or AStar single-source shortest path functions.
type Shortest[Node comparable] struct {
	// from holds the source node given to the function that returned
	// the Shortest value.
	from Node

	// nodes hold the nodes of the analysed graph.
	nodes []Node

	// indexOf contains a mapping between the id-dense representation of
	// the graph and the potentially id-sparse nodes held in nodes.
	indexOf map[Node]int

	// dist and next represent the shortest paths between nodes.
	//
	// Indices into dist and next are mapped through indexOf.
	//
	// dist contains the distances from the from node for each node in
	// the graph.
	dist []float64

	// next contains the shortest-path tree of the graph. The index is a
	// linear mapping of to-dense-id.
	next []int

	// hasNegativeCycle indicates whether the Shortest includes a
	// negative cycle. This should be set by the function that returned
	// the Shortest value.
	hasNegativeCycle bool

	// negCosts holds negative costs between pairs of nodes to report
	// negative cycles. negCosts must be initialised by routines that
	// can handle negative edge weights.
	negCosts map[negEdge]float64
}

// newShortestFrom returns a shortest path tree for paths from u
// initialised with the given nodes. The nodes held by the returned
// Shortest may be lazily added.
func newShortestFrom[Node comparable](u Node, nodes []Node) Shortest[Node] {
	indexOf := make(map[Node]int, len(nodes))
	for i, n := range nodes {
		indexOf[n] = i
	}

	p := Shortest[Node]{
		from: u,

		nodes:   nodes,
		indexOf: indexOf,

		dist: make([]float64, len(nodes)),
		next: make([]int, len(nodes)),
	}
	for i := range nodes {
		p.dist[i] = math.Inf(1)
		p.next[i] = -1
	}
	p.dist[indexOf[u]] = 0

	return p
}

// add adds a node to the Shortest, initialising its stored index and returning, and
// setting the distance and position as unconnected. add will panic if the node is
// already present.
func (p *Shortest[Node]) add(u Node) int {
	if _, exists := p.indexOf[u]; exists {
		panic("shortest: adding existing node")
	}
	idx := len(p.nodes)
	p.indexOf[u] = idx
	p.nodes = append(p.nodes, u)
	p.dist = append(p.dist, math.Inf(1))
	p.next = append(p.next, -1)
	return idx
}

// set sets the weight of the path from the node in p.nodes indexed by mid to the node
// indexed by to.
func (p Shortest[Node]) set(to int, weight float64, mid int) {
	p.dist[to] = weight
	p.next[to] = mid
	if weight < 0 {
		e := negEdge{from: mid, to: to}
		c, ok := p.negCosts[e]
		if !ok {
			p.negCosts[e] = weight
		} else if weight < c {
			// The only ways that we can have a new weight that is
			// lower than the previous weight is if either the edge
			// has already been traversed in a negative cycle, or
			// the edge is reachable from a negative cycle.
			// Either way the reported path is returned with a
			// negative infinite path weight.
			p.negCosts[e] = math.Inf(-1)
		}
	}
}

// From returns the starting node of the paths held by the Shortest.
func (p Shortest[Node]) From() Node { return p.from }

// WeightTo returns the weight of the minimum path to v. If the path to v includes
// a negative cycle, the returned weight will not reflect the true path weight.
func (p Shortest[Node]) WeightTo(v Node) float64 {
	to, toOK := p.indexOf[v]
	if !toOK {
		return math.Inf(1)
	}
	return p.dist[to]
}

// To returns a shortest path to v and the weight of the path. If the path
// to v includes a negative cycle, one pass through the cycle will be included
// in path, but any path leading into the negative cycle will be lost, and
// weight will be returned as -Inf.
func (p Shortest[Node]) To(v Node) (path []Node, weight float64) {
	to, toOK := p.indexOf[v]
	if !toOK || math.IsInf(p.dist[to], 1) {
		return nil, math.Inf(1)
	}
	from := p.indexOf[p.from]
	path = []Node{p.nodes[to]}
	weight = math.Inf(1)
	if p.hasNegativeCycle {
		seen := make(map[int]bool)
		seen[from] = true
		for to != from {
			next := p.next[to]
			if math.IsInf(p.negCosts[negEdge{from: next, to: to}], -1) {
				weight = math.Inf(-1)
			}
			if seen[to] {
				break
			}
			seen[to] = true
			path = append(path, p.nodes[next])
			to = next
		}
	} else {
		n := len(p.nodes)
		for to != from {
			to = p.next[to]
			path = append(path, p.nodes[to])
			if n < 0 {
				panic("path: unexpected negative cycle")
			}
			n--
		}
	}
	slices.Reverse(path)
	return path, math.Min(weight, p.dist[p.indexOf[v]])
}

// negEdge is a key into the negative costs map used by Shortest.
type negEdge struct{ from, to int }
