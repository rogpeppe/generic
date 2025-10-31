// Package mermaid provides functionality for marshaling graph structures
// to Mermaid diagram format. Mermaid is a text-based diagramming tool that
// generates diagrams from markdown-like syntax.
package mermaid

import (
	"bytes"
	"fmt"

	"github.com/rogpeppe/generic/graph"
)

// Marshaler represents a type that can be marshaled into Mermaid diagram format.
type Marshaler interface {
	// MarshalMermaid returns the Mermaid representation of the object.
	// It returns an error if the marshaling fails.
	MarshalMermaid() ([]byte, error)
}

// NewGraph creates a Marshaler from a GraphInterface. The resulting Marshaler
// can be used to generate a Mermaid graph diagram representation.
func NewGraph[Node comparable, Edge any](g GraphInterface[Node, Edge]) Marshaler {
	return &graphImpl[Node, Edge]{g}
}

// GraphInterface defines the interface required for a graph to be marshaled
// to Mermaid format. It extends the standard graph.Graph interface with
// methods to retrieve all nodes and node metadata.
type GraphInterface[Node comparable, Edge any] interface {
	graph.Graph[Node, Edge]
	// AllNodes returns all nodes in the graph.
	AllNodes() []Node
	// NodeInfo returns metadata about a node, including its ID, display text, and style.
	NodeInfo(Node) NodeInfo
}

// NodeInfo contains metadata about a graph node for Mermaid rendering.
type NodeInfo struct {
	// ID is the unique identifier for the node in the Mermaid diagram.
	ID string
	// Text is the display text for the node. If empty, ID is used instead.
	Text string
	// Style contains Mermaid style declarations for the node (e.g., "fill:#f9f,stroke:#333").
	Style string
}

type graphImpl[Node comparable, Edge any] struct {
	g GraphInterface[Node, Edge]
}

func (g *graphImpl[Node, Edge]) MarshalMermaid() ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "graph TD\n")
	for _, n := range g.g.AllNodes() {
		info := g.g.NodeInfo(n)
		if info.ID != info.Text && info.Text != "" {
			fmt.Fprintf(&buf, "  %s[%s]\n", info.ID, info.Text)
		}
		if info.Style != "" {
			fmt.Fprintf(&buf, "  style %s %s\n", info.ID, info.Style)
		}
		edges, ok := g.g.EdgesFrom(n)
		if ok {
			for _, e := range edges {
				from, to := g.g.Nodes(e)
				fmt.Fprintf(&buf, "  %s-->%s\n", g.g.NodeInfo(from).ID, g.g.NodeInfo(to).ID)
			}
		}
	}
	return buf.Bytes(), nil
}
