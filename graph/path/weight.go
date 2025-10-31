// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package path

import (
	"github.com/rogpeppe/generic/graph"
)

// Weighting is a mapping between a pair of nodes and a weight. It follows the
// semantics of the Weighter interface.
type Weighting[Edge any] func(e Edge) float64

// UniformCost returns a Weighting that returns an edge cost of 1 for existing
// edges, zero for node identity and Inf for otherwise absent edges.
func UniformCost[Node comparable, Edge any](g graph.Graph[Node, Edge]) Weighting[Edge] {
	return func(e Edge) float64 {
		from, to := g.Nodes(e)
		if from == to {
			return 0
		}
		return 1
	}
}

// Heuristic returns an estimate of the cost of travelling between two nodes.
type Heuristic[Node comparable] func(x, y Node) float64

// HeuristicCoster can be implemented by a [graph.Graph] to
// provide a default cost heuristic for the graph.
type HeuristicCoster[Node comparable] interface {
	HeuristicCost(x, y Node) float64
}
