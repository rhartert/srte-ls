package srte

// Edge represents an edge between two nodes in a directed graph.
type Edge struct {
	From int
	To   int
	Cost int
}

// Topology represents the topology of a network as a directed graph.
type Topology struct {
	Nexts [][]int
	Edges []Edge
}

// NewTopology creates a new topology with the specified edges and number of
// nodes. It is important to ensure that edges are only between nodes within
// the range [0, nNodes); otherwise, the function will panic.
func NewTopology(edges []Edge, nNodes int) *Topology {
	dg := &Topology{
		Nexts: make([][]int, nNodes),
		Edges: make([]Edge, len(edges)),
	}
	for i, e := range edges {
		dg.Edges[i] = e
		dg.Nexts[e.From] = append(dg.Nexts[e.From], i)
	}
	return dg
}
