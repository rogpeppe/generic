package mermaid

import (
	"cmp"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestNewGraph(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")

	m := NewGraph(g)
	qt.Assert(t, qt.IsNotNil(m))
}

func TestMarshalMermaid_EmptyGraph(t *testing.T) {
	g := newTestGraph()
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n"))
}

func TestMarshalMermaid_SingleNode(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n"))
}

func TestMarshalMermaid_SingleNode_NoText(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "", "")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	// When text is empty, no node declaration should be output
	qt.Assert(t, qt.Equals(string(result), "graph TD\n"))
}

func TestMarshalMermaid_SingleNode_SameIDAndText(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "A", "")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	// When ID and text are the same, no node declaration should be output
	qt.Assert(t, qt.Equals(string(result), "graph TD\n"))
}

func TestMarshalMermaid_NodeWithStyle(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "fill:#f9f,stroke:#333")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n  style A fill:#f9f,stroke:#333\n"))
}

func TestMarshalMermaid_SimpleEdge(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")
	g.addNode("B", "Node B", "")
	g.addEdge("A", "B")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n  A-->B\n  B[Node B]\n"))
}

func TestMarshalMermaid_MultipleEdges(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")
	g.addNode("B", "Node B", "")
	g.addNode("C", "Node C", "")
	g.addEdge("A", "B")
	g.addEdge("A", "C")
	g.addEdge("B", "C")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n  A-->B\n  A-->C\n  B[Node B]\n  B-->C\n  C[Node C]\n"))
}

func TestMarshalMermaid_ComplexGraph(t *testing.T) {
	g := newTestGraph()
	g.addNode("start", "Start", "fill:#9f9,stroke:#333")
	g.addNode("process", "Process Data", "fill:#99f,stroke:#333")
	g.addNode("decision", "Valid?", "fill:#ff9,stroke:#333")
	g.addNode("endNode", "End", "fill:#f99,stroke:#333")

	g.addEdge("start", "process")
	g.addEdge("process", "decision")
	g.addEdge("decision", "endNode")
	g.addEdge("decision", "process")

	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))

	expected := "graph TD\n" +
		"  start[Start]\n  style start fill:#9f9,stroke:#333\n  start-->process\n" +
		"  process[Process Data]\n  style process fill:#99f,stroke:#333\n  process-->decision\n" +
		"  decision[Valid?]\n  style decision fill:#ff9,stroke:#333\n  decision-->endNode\n  decision-->process\n" +
		"  endNode[End]\n  style endNode fill:#f99,stroke:#333\n"

	qt.Assert(t, qt.Equals(string(result), expected))
}

func TestMarshalMermaid_EdgesWithNoCustomText(t *testing.T) {
	g := newTestGraph()
	// Nodes with IDs only (no custom text)
	g.addNode("A", "A", "")
	g.addNode("B", "B", "")
	g.addEdge("A", "B")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A-->B\n"))
}

func TestMarshalMermaid_SelfLoop(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")
	g.addEdge("A", "A")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n  A-->A\n"))
}

func TestMarshalMermaid_NodeWithStyleNoText(t *testing.T) {
	g := newTestGraph()
	g.addNode("X", "X", "fill:#abc")
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	// Even when ID equals text, style should still be output
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  style X fill:#abc\n"))
}

func TestMarshalMermaid_IsolatedNodes(t *testing.T) {
	g := newTestGraph()
	g.addNode("A", "Node A", "")
	g.addNode("B", "Node B", "")
	g.addNode("C", "Node C", "")
	// No edges
	m := NewGraph(g)

	result, err := m.MarshalMermaid()
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(string(result), "graph TD\n  A[Node A]\n  B[Node B]\n  C[Node C]\n"))
}

// testGraph implements GraphInterface for testing
type testGraph struct {
	nodes []string
	edges map[string][]testEdge
	info  map[string]NodeInfo
}

type testEdge struct {
	from, to string
}

func newTestGraph() *testGraph {
	return &testGraph{
		nodes: []string{},
		edges: make(map[string][]testEdge),
		info:  make(map[string]NodeInfo),
	}
}

func (g *testGraph) addNode(id string, text string, style string) {
	g.nodes = append(g.nodes, id)
	g.info[id] = NodeInfo{
		ID:    id,
		Text:  text,
		Style: style,
	}
}

func (g *testGraph) addEdge(from, to string) {
	edge := testEdge{from: from, to: to}
	g.edges[from] = append(g.edges[from], edge)
}

func (g *testGraph) AllNodes() []string {
	return g.nodes
}

func (g *testGraph) NodeInfo(n string) NodeInfo {
	return g.info[n]
}

func (g *testGraph) EdgesFrom(n string) ([]testEdge, bool) {
	edges, ok := g.edges[n]
	return edges, ok
}

func (g *testGraph) Nodes(e testEdge) (string, string) {
	return e.from, e.to
}

func (g *testGraph) CmpNode(n0, n1 string) int {
	return cmp.Compare(n0, n1)
}
