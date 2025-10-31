// Copyright ©2014 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package path

import (
	"github.com/rogpeppe/generic/heap"

	"github.com/rogpeppe/generic/graph"
)

// AStar finds the A*-shortest path from s to t in g using the heuristic h. The path and
// its cost are returned in a Shortest along with paths and costs to all nodes explored
// during the search. The number of expanded nodes is also returned. This value may help
// with heuristic tuning.
//
// The path will be the shortest path if the heuristic is admissible. A heuristic is
// admissible if for any node, n, in the graph, the heuristic estimate of the cost of
// the path from n to t is less than or equal to the true cost of that path.
//
// If h is nil, AStar will use the g.HeuristicCost method if g implements HeuristicCoster,
// falling back to NullHeuristic otherwise. If the graph does not implement Weighted,
// UniformCost is used. AStar will panic if g has an A*-reachable negative edge weight.
func AStar[Node comparable, Edge any](s, t Node, g graph.Graph[Node, Edge], h Heuristic[Node]) (path Shortest[Node], expanded int) {
	if !graph.NodeInGraph(g, s) || !graph.NodeInGraph(g, t) {
		return Shortest[Node]{from: s}, 0
	}
	var weight Weighting[Edge]
	if wg, ok := g.(graph.Weighted[Node, Edge]); ok {
		weight = wg.EdgeWeight
	} else {
		weight = UniformCost(g)
	}
	if h == nil {
		if g, ok := g.(HeuristicCoster[Node]); ok {
			h = g.HeuristicCost
		} else {
			h = NullHeuristic
		}
	}

	path = newShortestFrom(s, []Node{s, t})

	visited := make(map[Node]bool)
	open := newAStarQueue[Node]()
	open.push(&aStarNode[Node]{node: s, gscore: 0, fscore: h(s, t)})

	for open.heap.Len() != 0 {
		u := open.pop()
		i := path.indexOf[u.node]
		expanded++

		if u.node == t {
			break
		}

		visited[u.node] = true
		edges, _ := g.EdgesFrom(u.node)
		for _, e := range edges {
			_, v := g.Nodes(e)
			if visited[v] {
				continue
			}
			j, ok := path.indexOf[v]
			if !ok {
				j = path.add(v)
			}

			w := weight(e)
			if w < 0 {
				panic("path: A* negative edge weight")
			}
			g := u.gscore + w
			if n, ok := open.node(v); !ok {
				path.set(j, g, i)
				open.push(&aStarNode[Node]{node: v, gscore: g, fscore: g + h(v, t)})
			} else if g < n.gscore {
				path.set(j, g, i)
				open.update(v, g, g+h(v, t))
			}
		}
	}

	return path, expanded
}

// NullHeuristic is an admissible, consistent heuristic that will not speed up computation.
func NullHeuristic[Node any](_, _ Node) float64 {
	return 0
}

// aStarNode adds A* accounting to a graph Node.
type aStarNode[Node any] struct {
	node      Node
	gscore    float64
	fscore    float64
	heapIndex int
}

// aStarQueue is an A* priority queue.
type aStarQueue[Node comparable] struct {
	heap   *heap.Heap[*aStarNode[Node]]
	byNode map[Node]*aStarNode[Node]
}

func newAStarQueue[Node comparable]() *aStarQueue[Node] {
	return &aStarQueue[Node]{
		heap: heap.New[*aStarNode[Node]](nil, func(n0, n1 *aStarNode[Node]) bool {
			return n0.fscore < n1.fscore
		}, func(e **aStarNode[Node], i int) {
			(*e).heapIndex = i
		}),
		byNode: make(map[Node]*aStarNode[Node]),
	}
}

func (q *aStarQueue[Node]) push(n *aStarNode[Node]) {
	q.heap.Push(n)
	q.byNode[n.node] = n
}

func (q *aStarQueue[Node]) pop() *aStarNode[Node] {
	n := q.heap.Pop()
	delete(q.byNode, n.node)
	return n
}

func (q *aStarQueue[Node]) update(n Node, g, f float64) {
	an, ok := q.byNode[n]
	if !ok {
		return
	}
	an.gscore = g
	an.fscore = f
	q.heap.Fix(an.heapIndex)
}

func (q *aStarQueue[Node]) node(n Node) (*aStarNode[Node], bool) {
	an, ok := q.byNode[n]
	return an, ok
}
