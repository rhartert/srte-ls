package srte

import (
	"fmt"
	"math"
	"sort"

	"github.com/rhartert/yagh"
)

// EdgeRatio represent an edge in a forwarding graph and the ratio of load sent
// on that edge. For example, EdgeRatio{5, 0.5} means that 50% of the load sent
// on the forwarding graph traverses edge 5.
type EdgeRatio struct {
	Edge  int
	Ratio float64
}

type FGraphs struct {
	edgesRatios [][][]EdgeRatio
}

// EdgeRatios returns the list of EdgeRatio pairs on the forwarding graph from
// node s to node t.
func (fgs *FGraphs) EdgeRatios(s int, t int) []EdgeRatio {
	return fgs.edgesRatios[s][t]
}

func NewFGraphs(g *Digraph) (*FGraphs, error) {
	nNodes := len(g.Nexts)

	fgs := &FGraphs{
		edgesRatios: make([][][]EdgeRatio, nNodes),
	}

	for u := 0; u < nNodes; u++ {
		fgs.edgesRatios[u] = make([][]EdgeRatio, nNodes)

		prevs, err := shortestDAG(g, u)
		if err != nil {
			return nil, err
		}

		for v := 0; v < nNodes; v++ {
			if u == v {
				continue
			}
			edgeRatiosMap := forwardingGraph(g, prevs, u, v)
			fgs.edgesRatios[u][v] = make([]EdgeRatio, 0, len(edgeRatiosMap))
			for e, r := range edgeRatiosMap {
				fgs.edgesRatios[u][v] = append(fgs.edgesRatios[u][v], EdgeRatio{
					Edge:  e,
					Ratio: r,
				})
			}
			sort.Slice(fgs.edgesRatios[u][v], func(i, j int) bool {
				return fgs.edgesRatios[u][v][i].Edge < fgs.edgesRatios[u][v][j].Edge
			})
		}
	}

	return fgs, nil
}

// foardingGraph computes the fraction of load sent on each edge when sending
// traffic from node s to node t.
//
// The returned load must respect the following invariants where loadIn[n] is
// the total amount of load on edges reaching node n and loadOut[n] is the total
// amount of load on edges leaving n:
//   - loadIn[s] = 1 and loadOut[s] = 0,
//   - loadIn[t] = 0 and loadOut[t] = 1,
//   - loadIn[n] = loadOut[n] for all node n != s, t.
//
// The algorithm operates in two phase. The first phase traverses prevs
// from t to s to compute the DAG of all the shortest paths from s to t.
// The second phase traverses that DAG to compute the fraction of load sent
// over each edge from s to t.
//
// Compute the fraction of traffic sent on each edge. For any edge (u, v),
// the total fraction of traffic received at node u must be computed before
// computing the fraction sent on the edge. This is done by processing the
// nodes in their topological order.
func forwardingGraph(g *Digraph, prevs [][]int, s int, t int) map[int]float64 {
	queue := []int{} // used by both steps below
	nNodes := len(g.Nexts)

	// Step 1: extract DAG
	// -------------------
	nexts := make([][]int, nNodes)
	degrees := make([]int, nNodes)

	queue = append(queue, t)
	inQueue := make([]bool, nNodes)
	inQueue[t] = true

	for i := 0; i < len(queue); i++ {
		v := queue[i]
		degrees[v] = len(prevs[v])
		for _, e := range prevs[v] {
			u := g.Edges[e].From
			if !inQueue[u] {
				queue = append(queue, u)
				inQueue[u] = true
			}
			nexts[u] = append(nexts[u], e)
		}
	}

	// Step 2: Compute load ratios
	// ---------------------------
	nodeLoad := make([]float64, nNodes)
	edgeLoad := make(map[int]float64) // result

	queue = queue[:0] // reset
	queue = append(queue, s)
	nodeLoad[s] = 1.0
	for i := 0; i < len(queue); i++ {
		u := queue[i]
		for _, e := range nexts[u] {
			v := g.Edges[e].To

			l := nodeLoad[u] / float64(len(nexts[u]))
			edgeLoad[e] = l
			nodeLoad[v] += l

			degrees[v] -= 1
			if degrees[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	return edgeLoad
}

// shortestDAG computes and returns a DAG that encapsulates the shortest paths
// from a specified source node src to all other nodes within the digraph g.
//
// This function returns a slice that maps each node v in the graph o a list of
// incoming edges (u, v), where each edge represents a part of the shortest path
// from src to v. If a node v is unreachable from src, its corresponding list
// will be empty.
func shortestDAG(g *Digraph, src int) ([][]int, error) {
	if g == nil {
		return nil, fmt.Errorf("digraph is nil")
	}

	nNodes := len(g.Nexts)
	if src < 0 || nNodes <= src {
		return nil, fmt.Errorf("node %d is not in the graph", src)
	}

	prevs := make([][]int, nNodes)
	costs := make([]int, nNodes)
	for i := range costs {
		costs[i] = math.MaxInt
	}

	h := yagh.New[int](nNodes)
	h.Put(src, 0)
	costs[src] = 0

	for h.Size() > 0 {
		entry := h.Pop()
		u, c := entry.Elem, entry.Cost

		for _, e := range g.Nexts[u] {
			newCost := c + g.Edges[e].Cost
			v := g.Edges[e].To

			// Path src -> u -> v is worse than the best known path.
			if costs[v] < newCost {
				continue
			}

			// Path src -> u -> v is one of the best paths to v so far.
			if costs[v] == newCost {
				prevs[v] = append(prevs[v], e)
				continue
			}

			// Path src -> u -> v is the better than the best path to v so far.
			costs[v] = newCost
			prevs[v] = []int{e}
			h.Put(v, newCost)
		}
	}

	return prevs, nil
}
