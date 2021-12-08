// Copyright 2015 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This code is derived from the code in the v.io/x/lib/toposort package.

package graph

// TopoSort returns the topologically sorted nodes, along with some of the cycles
// (if any) that were encountered.  You're guaranteed that len(cycles)==0 iff
// there are no cycles in the graph, otherwise an arbitrary (but non-empty) list
// of cycles is returned.
//
// If there are cycles the sorting is best-effort; portions of the graph that
// are acyclic will still be ordered correctly, and the cyclic portions have an
// arbitrary ordering.
//
// Sort is deterministic; given the same sequence of inputs it always returns
// the same output, even if the inputs are only partially ordered.
func TopoSort[Node comparable, Edge any](g Graph[Node, Edge]) (sorted []Node, cycles [][]Node) {
	// The strategy is the standard simple approach of performing DFS on the
	// graph.  Details are outlined in the above wikipedia article.
	v := &visitor[Node, Edge]{
		g:    g,
		done: make(map[Node]bool),
	}
	for _, n := range g.AllNodes() {
		v.visiting = make(map[Node]bool)
		cycles = append(cycles, v.visit(n)...)
	}
	return v.sorted, cycles
}

type visitor[Node comparable, Edge any] struct {
	g        Graph[Node, Edge]
	done     map[Node]bool
	visiting map[Node]bool
	sorted   []Node
}

// visit performs depth-first search on the graph and fills in sorted and cycles as it
// traverses.  We use done to indicate a node has been fully explored, and
// visiting to indicate a node is currently being explored.
//
// The cycle collection strategy is to wait until we've hit a repeated node in
// visiting, and add that node to cycles and return.  Thereafter as the
// recursive stack is unwound, nodes append themselves to the end of each cycle,
// until we're back at the repeated node.  This guarantees that if the graph is
// cyclic we'll return at least one of the cycles.
func (v *visitor[Node, Edge]) visit(n Node) (cycles [][]Node) {
	if v.done[n] {
		return nil
	}
	if v.visiting[n] {
		return [][]Node{{n}}
	}
	v.visiting[n] = true
	for _, edge := range v.g.Edges(n) {
		_, child := v.g.Nodes(edge)
		cycles = append(cycles, v.visit(child)...)
	}
	v.done[n] = true
	v.sorted = append(v.sorted, n)
	// Update cycles.  If it's empty none of our children detected any cycles, and
	// there's nothing to do.  Otherwise we append ourselves to the cycle, iff the
	// cycle hasn't completed yet.  We know the cycle has completed if the first
	// and last item in the cycle are the same, with an exception for the single
	// item case; self-cycles are represented as the same node appearing twice.
	for cx := range cycles {
		len := len(cycles[cx])
		if len == 1 || cycles[cx][0] != cycles[cx][len-1] {
			cycles[cx] = append(cycles[cx], n)
		}
	}
	return cycles
}

// DumpCycles dumps the cycles returned from TopoSort, using toString to
// convert each node into a string.
func DumpCycles[Node any](cycles [][]Node, toString func(n Node) string) string {
	var str string
	for cyclex, cycle := range cycles {
		if cyclex > 0 {
			str += " "
		}
		str += "["
		for nodex, node := range cycle {
			if nodex > 0 {
				str += " <= "
			}
			str += toString(node)
		}
		str += "]"
	}
	return str
}
