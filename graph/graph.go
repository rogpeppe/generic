package graph

type Graph[Node comparable, Edge any] interface {
	Edges(n Node) []Edge
	Nodes(e Edge) (from, to Node)
}

// item holds an item in the node fringe being calculated by
// ShortestPath. We might normally declare this inside ShortestPath
// itself, but that would mean we couldn't refer to it globally in the
// generated code (not a problem when doing generics directly in the
// compiler)
type item[Node, Edge any] struct {
	n     Node
	dist  int
	index int
	edge  Edge
}

func ShortestPath[Node comparable, Edge any](g Graph[Node, Edge], from, to Node) []Edge {
	type itemNE = item[Node, Edge]
	h := NewHeap([]*itemNE{{
		n:     from,
		dist:  0,
		index: 0,
	}}, func(i1, i2 *itemNE) bool {
		return i1.dist < i2.dist
	}, func(it **itemNE, i int) {
		(*it).index = i
	})
	// Note: we'd like to use map[Node] *item but we
	// can't do that since we can't provide the equality and
	// hash functions to the internal map implementation.
	nodes := make(map[Node]*itemNE)
	var found *itemNE
	for len(h.Items) > 0 {
		nearest := h.Pop()
		if nearest.n == to {
			found = nearest
			break
		}
		for _, e := range g.Edges(nearest.n) {
			edgeFrom, edgeTo := g.Nodes(e)
			if edgeFrom != nearest.n {
				continue
			}
			dist := nearest.dist + 1 // Could use e.Length() instead of 1.
			toItem, ok := nodes[edgeTo]
			if !ok {
				it := &itemNE{
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
	return edges
}
