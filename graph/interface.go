package graph

type Graph[Node comparable, Edge any] interface {
	Edges(n Node) []Edge
	Nodes(e Edge) (from, to Node)
	AllNodes() []Node
}
