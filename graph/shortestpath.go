package graph

import "github.com/rogpeppe/generic/heap"

// item holds an item in the node fringe being calculated by
// ShortestPath. We might normally declare this inside ShortestPath
// itself, but local type declarations inside generic functions
// aren't currently supported.
type item[Node, Edge any] struct {
	n     Node
	dist  int
	index int
	edge  Edge
}

// ShortestPath returns the shortest path from -> to in the graph g
// using Dijkstra's algorithm. The returned slice holds all the edges
// leading from the source to the destination.
func ShortestPath[Node comparable, Edge any](g Graph[Node, Edge], from, to Node) []Edge {
	h := heap.New([]*item[Node, Edge]{{
		n:     from,
		dist:  0,
		index: 0,
	}}, func(i1, i2 *item[Node, Edge]) bool {
		return i1.dist < i2.dist
	}, func(it **item[Node, Edge], i int) {
		(*it).index = i
	})
	nodes := make(map[Node]*item[Node, Edge])
	var found *item[Node, Edge]
	for len(h.Items) > 0 {
		nearest := h.Pop()
		if nearest.n == to {
			found = nearest
			break
		}
		edges, _ := g.EdgesFrom(nearest.n)
		for _, e := range edges {
			edgeFrom, edgeTo := g.Nodes(e)
			if edgeFrom != nearest.n {
				continue
			}
			dist := nearest.dist + 1 // Could use e.Length() instead of 1 if edges had lengths.
			toItem, ok := nodes[edgeTo]
			if !ok {
				it := &item[Node, Edge]{
					n:    edgeTo,
					dist: dist,
					edge: e,
				}
				nodes[edgeTo] = it
				h.Push(it)
			} else if dist < toItem.dist {
				toItem.dist = dist
				toItem.edge = e
				h.Fix(toItem.index)
			}
		}
	}
	if found == nil {
		return nil
	}
	var edges []Edge
	for {
		edges = append(edges, found.edge)
		edgeFrom, _ := g.Nodes(found.edge)
		if edgeFrom == from {
			break
		}
		found = nodes[edgeFrom]
	}
	reverse(edges)
	return edges
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
