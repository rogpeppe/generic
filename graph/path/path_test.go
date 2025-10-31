package path

import (
	"cmp"
	"iter"
	"math"
	"slices"
	"testing"

	"github.com/go-quicktest/qt"
)

// testGraph is a simple weighted graph for testing.
type testGraph struct {
	edges  map[string][]edge
	nodes  []string
	weight func(edge) float64
}

type edge struct {
	from, to string
	weight   float64
}

func (g *testGraph) CmpNode(n0, n1 string) int {
	return cmp.Compare(n0, n1)
}

func (g *testGraph) AllNodes() iter.Seq[string] {
	return slices.Values(g.nodes)
}

func (g *testGraph) EdgesFrom(n string) ([]edge, bool) {
	edges, ok := g.edges[n]
	return edges, ok
}

func (g *testGraph) Nodes(e edge) (from, to string) {
	return e.from, e.to
}

func (g *testGraph) EdgeWeight(e edge) float64 {
	if g.weight != nil {
		return g.weight(e)
	}
	return e.weight
}

// newTestGraph creates a test graph with the given edges.
func newTestGraph(edges []edge) *testGraph {
	g := &testGraph{
		edges: make(map[string][]edge),
	}
	nodeSet := make(map[string]bool)
	for _, e := range edges {
		g.edges[e.from] = append(g.edges[e.from], e)
		nodeSet[e.from] = true
		nodeSet[e.to] = true
	}
	// Ensure all nodes are in the edges map, even if they have no outgoing edges
	for n := range nodeSet {
		if _, ok := g.edges[n]; !ok {
			g.edges[n] = nil
		}
		g.nodes = append(g.nodes, n)
	}
	slices.Sort(g.nodes)
	return g
}

func TestAStarBasicPath(t *testing.T) {
	// Simple linear graph: A -> B -> C -> D
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"B", "C", 1},
		{"C", "D", 1},
	})

	path, expanded := AStar("A", "D", g, nil)
	qt.Assert(t, qt.Equals(expanded, 4))
	qt.Assert(t, qt.Equals(path.From(), "A"))

	// Check weight to D
	weight := path.WeightTo("D")
	qt.Assert(t, qt.Equals(weight, 3.0))

	// Check the path
	nodes, pathWeight := path.To("D")
	qt.Assert(t, qt.Equals(pathWeight, 3.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B", "C", "D"}))
}

func TestAStarShortestPath(t *testing.T) {
	// Graph with multiple paths, one shorter:
	//     1
	// A -----> B
	// |        |
	// |5       |1
	// v        v
	// C -----> D
	//     1
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"B", "D", 1},
		{"A", "C", 5},
		{"C", "D", 1},
	})

	path, _ := AStar("A", "D", g, nil)

	// Should take A -> B -> D (weight 2) not A -> C -> D (weight 6)
	nodes, weight := path.To("D")
	qt.Assert(t, qt.Equals(weight, 2.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B", "D"}))
}

func TestAStarComplexGraph(t *testing.T) {
	// More complex graph:
	//      2        3
	//  A ----> B -----> D
	//  |       |        ^
	//  |1      |1       |2
	//  v       v        |
	//  C ----> E -------+
	//      1
	g := newTestGraph([]edge{
		{"A", "B", 2},
		{"A", "C", 1},
		{"B", "D", 3},
		{"B", "E", 1},
		{"C", "E", 1},
		{"E", "D", 2},
	})

	path, _ := AStar("A", "D", g, nil)

	// Shortest path: A -> C -> E -> D (weight 4)
	nodes, weight := path.To("D")
	qt.Assert(t, qt.Equals(weight, 4.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "C", "E", "D"}))
}

func TestAStarNoPath(t *testing.T) {
	// Disconnected graph
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"C", "D", 1},
	})

	path, expanded := AStar("A", "D", g, nil)
	qt.Assert(t, qt.Equals(expanded, 2)) // Only explores A and B

	// Weight should be infinity
	weight := path.WeightTo("D")
	qt.Assert(t, qt.IsTrue(math.IsInf(weight, 1)))

	// Path should be nil
	nodes, pathWeight := path.To("D")
	qt.Assert(t, qt.IsNil(nodes))
	qt.Assert(t, qt.IsTrue(math.IsInf(pathWeight, 1)))
}

func TestAStarSameStartAndEnd(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"B", "C", 1},
	})

	path, expanded := AStar("A", "A", g, nil)
	qt.Assert(t, qt.Equals(expanded, 1)) // Only examines start node

	weight := path.WeightTo("A")
	qt.Assert(t, qt.Equals(weight, 0.0))

	nodes, pathWeight := path.To("A")
	qt.Assert(t, qt.Equals(pathWeight, 0.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A"}))
}

func TestAStarNonExistentNode(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", 1},
	})

	// Start node doesn't exist
	path, expanded := AStar("Z", "B", g, nil)
	qt.Assert(t, qt.Equals(expanded, 0))
	qt.Assert(t, qt.Equals(path.From(), "Z"))

	// End node doesn't exist (returns early with no expansions)
	path, expanded = AStar("A", "Z", g, nil)
	qt.Assert(t, qt.Equals(expanded, 0)) // Returns early without exploring
	weight := path.WeightTo("Z")
	qt.Assert(t, qt.IsTrue(math.IsInf(weight, 1)))
}

func TestAStarWithCustomHeuristic(t *testing.T) {
	// Graph representing a grid where heuristic can help
	//   A --1-- B --1-- C
	//   |       |       |
	//   1       1       1
	//   |       |       |
	//   D --1-- E --1-- F
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"B", "C", 1},
		{"A", "D", 1},
		{"B", "E", 1},
		{"C", "F", 1},
		{"D", "E", 1},
		{"E", "F", 1},
	})

	// Heuristic that estimates distance (admissible)
	heuristic := func(from, to string) float64 {
		// Simple heuristic based on string comparison
		// This is admissible because it never overestimates
		if from == to {
			return 0
		}
		return 1 // Underestimate of remaining distance
	}

	path, _ := AStar("A", "F", g, heuristic)

	// Should find a valid path
	nodes, weight := path.To("F")
	qt.Assert(t, qt.Equals(weight, 3.0)) // Either A->B->C->F or A->D->E->F
	qt.Assert(t, qt.Equals(len(nodes), 4))
	qt.Assert(t, qt.Equals(nodes[0], "A"))
	qt.Assert(t, qt.Equals(nodes[3], "F"))
}

func TestAStarNullHeuristic(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", 2},
		{"B", "C", 3},
	})

	// Explicitly use null heuristic (same as nil)
	path, _ := AStar("A", "C", g, NullHeuristic[string])

	nodes, weight := path.To("C")
	qt.Assert(t, qt.Equals(weight, 5.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B", "C"}))
}

func TestAStarWithUniformCost(t *testing.T) {
	// Simple graph without weights
	g := &simpleGraphAdapter{
		edges: map[string][]string{
			"A": {"B", "C"},
			"B": {"D"},
			"C": {"D"},
		},
	}

	path, _ := AStar("A", "D", g, nil)

	// With uniform cost, both paths have same weight
	weight := path.WeightTo("D")
	qt.Assert(t, qt.Equals(weight, 2.0)) // 2 edges
}

// simpleGraphAdapter adapts a simple graph to the Graph interface
type simpleGraphAdapter struct {
	edges map[string][]string
}

type simpleEdge struct{ from, to string }

func (g *simpleGraphAdapter) CmpNode(n0, n1 string) int {
	return cmp.Compare(n0, n1)
}

func (g *simpleGraphAdapter) EdgesFrom(n string) ([]simpleEdge, bool) {
	to, exists := g.edges[n]
	if !exists {
		// Check if this node exists as a destination in any edge
		for _, targets := range g.edges {
			for _, t := range targets {
				if t == n {
					return nil, true // Node exists but has no outgoing edges
				}
			}
		}
		return nil, false
	}
	edges := make([]simpleEdge, len(to))
	for i, t := range to {
		edges[i] = simpleEdge{n, t}
	}
	return edges, true
}

func (g *simpleGraphAdapter) Nodes(e simpleEdge) (from, to string) {
	return e.from, e.to
}

func TestAStarPanicsOnNegativeWeight(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", -1}, // Negative weight
		{"B", "C", 1},
	})

	qt.Assert(t, qt.PanicMatches(func() {
		AStar("A", "C", g, nil)
	}, "path: A\\* negative edge weight"))
}

func TestShortestWeightToIntermediateNodes(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", 2},
		{"B", "C", 3},
		{"C", "D", 1},
	})

	path, _ := AStar("A", "D", g, nil)

	// Check weights to intermediate nodes
	qt.Assert(t, qt.Equals(path.WeightTo("A"), 0.0))
	qt.Assert(t, qt.Equals(path.WeightTo("B"), 2.0))
	qt.Assert(t, qt.Equals(path.WeightTo("C"), 5.0))
	qt.Assert(t, qt.Equals(path.WeightTo("D"), 6.0))

	// Check path to intermediate node
	nodes, weight := path.To("B")
	qt.Assert(t, qt.Equals(weight, 2.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B"}))
}

func TestShortestToUnreachedNode(t *testing.T) {
	g := newTestGraph([]edge{
		{"A", "B", 1},
		{"C", "D", 1},
	})

	path, _ := AStar("A", "B", g, nil)

	// Node D was never reached
	weight := path.WeightTo("D")
	qt.Assert(t, qt.IsTrue(math.IsInf(weight, 1)))

	nodes, pathWeight := path.To("D")
	qt.Assert(t, qt.IsNil(nodes))
	qt.Assert(t, qt.IsTrue(math.IsInf(pathWeight, 1)))
}

func TestUniformCost(t *testing.T) {
	g := &simpleGraphAdapter{
		edges: map[string][]string{
			"A": {"B"},
		},
	}

	weighting := UniformCost[string, simpleEdge](g)

	// Self-edge should have cost 0
	qt.Assert(t, qt.Equals(weighting(simpleEdge{"A", "A"}), 0.0))

	// Regular edge should have cost 1
	qt.Assert(t, qt.Equals(weighting(simpleEdge{"A", "B"}), 1.0))
}

func TestNullHeuristic(t *testing.T) {
	// Null heuristic always returns 0
	qt.Assert(t, qt.Equals(NullHeuristic("A", "B"), 0.0))
	qt.Assert(t, qt.Equals(NullHeuristic("X", "Y"), 0.0))
	qt.Assert(t, qt.Equals(NullHeuristic(1, 100), 0.0))
}

func TestAStarLargerGraph(t *testing.T) {
	// Larger graph to test performance characteristics
	edges := []edge{
		{"A", "B", 1},
		{"A", "C", 4},
		{"B", "C", 2},
		{"B", "D", 5},
		{"C", "D", 1},
		{"C", "E", 3},
		{"D", "F", 2},
		{"E", "F", 1},
		{"E", "G", 4},
		{"F", "G", 1},
	}
	g := newTestGraph(edges)

	path, expanded := AStar("A", "G", g, nil)

	// Optimal path: A -> B -> C -> D -> F -> G (weight 7)
	// Not A -> B -> C -> E -> F -> G (weight 8)
	nodes, weight := path.To("G")
	qt.Assert(t, qt.Equals(weight, 7.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B", "C", "D", "F", "G"}))
	qt.Assert(t, qt.IsTrue(expanded > 0)) // Should explore some nodes
}

// heuristicGraph wraps a testGraph and provides a HeuristicCost method
type heuristicGraph struct {
	*testGraph
}

func (hg heuristicGraph) HeuristicCost(x, y string) float64 {
	// Simple heuristic
	if x == y {
		return 0
	}
	return 0.5 // Admissible underestimate
}

func TestHeuristicCosterInterface(t *testing.T) {
	// Test that a graph implementing HeuristicCoster uses its heuristic
	base := newTestGraph([]edge{
		{"A", "B", 1},
		{"B", "C", 1},
	})

	hg := heuristicGraph{base}

	// Should use the graph's HeuristicCost method when nil heuristic passed
	path, _ := AStar[string, edge]("A", "C", hg, nil)

	nodes, weight := path.To("C")
	qt.Assert(t, qt.Equals(weight, 2.0))
	qt.Assert(t, qt.DeepEquals(nodes, []string{"A", "B", "C"}))
}
