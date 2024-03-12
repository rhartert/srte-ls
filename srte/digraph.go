package srte

// Edge represents an edge between two nodes in a directed graph.
type Edge struct {
	From int
	To   int
	Cost int
}

// Digraph represents a directed graph.
type Digraph struct {
	Nexts [][]int
	Edges []Edge
}

// NewDigraph creates a new directed graph with the specified edges and number
// of nodes. It is important to ensure that edges are only between nodes within
// the range [0, nNodes); otherwise, the function will panic.
func NewDigraph(edges []Edge, nNodes int) *Digraph {
	dg := &Digraph{
		Nexts: make([][]int, nNodes),
		Edges: make([]Edge, len(edges)),
	}
	for i, e := range edges {
		dg.Edges[i] = e
		dg.Nexts[e.From] = append(dg.Nexts[e.From], i)
	}
	return dg
}
