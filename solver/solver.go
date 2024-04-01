// Package solver contains an implementation of the Link-Guided Search algorithm
// to minimize the maximum utilization of a network.
package solver

import (
	"log"
	"math"

	"github.com/rhartert/srte-ls/srte"
	"github.com/rhartert/yagh"
)

type Config struct {
	// Parameter alpha controls the likelihood of selecting an edge where the
	// likelihood P(e) of selecting edge e is determined by its utilization
	// raised to the power of alpha: P(e) = (util[e]^alpha) / Σ(util[j]^alpha).
	// High values of alpha increase the probability of selecting the most
	// utilized edge. By contrast, small values f alpha flatten the probability
	// distribution. In particular, setting alpha to zero results in random
	// uniform distribution.
	Alpha float64
}

type LinkGuidedSolver struct {
	state *srte.SRTE
	cfg   Config

	edgeWheel     *srte.SumTree
	edgesByUtil   *yagh.IntMap[float64]
	demandsByEdge []map[int]int64
}

// NewLinkGuidedSolver returns a new instance of a Link-Guided Search solver to
// minimize the maximum utilization of the given SRTE state.
func NewLinkGuidedSolver(state *srte.SRTE, cfg Config) *LinkGuidedSolver {
	nEdges := len(state.Instance.Graph.Edges)

	lgs := &LinkGuidedSolver{
		state:         state,
		cfg:           cfg,
		edgeWheel:     srte.NewSumTree(nEdges),
		edgesByUtil:   yagh.New[float64](nEdges),
		demandsByEdge: make([]map[int]int64, nEdges),
	}

	for e := 0; e < nEdges; e++ {
		lgs.edgeWheel.SetWeight(e, math.Pow(state.Utilization(e), cfg.Alpha))
		lgs.edgesByUtil.Put(e, -state.Utilization(e)) // non-decreasing order
		lgs.demandsByEdge[e] = map[int]int64{}
	}

	for i, d := range state.Instance.Demands {
		for _, er := range state.FGraphs.EdgeRatios(d.From, d.To) {
			lgs.demandsByEdge[er.Edge][i] = int64(er.Ratio * float64(d.Bandwidth))
		}
	}

	return lgs
}

// MostUtilizedEdge returns the ID of the edge with the highest utilization. If
// several edges have the same highest utilization, the one with the smallest
// ID is returned.
func (lgs *LinkGuidedSolver) MostUtilizedEdge() int {
	entry, _ := lgs.edgesByUtil.Min()
	return entry.Elem
}

// MaxUtilization returns the maximum edge utilization.
func (lgs *LinkGuidedSolver) MaxUtilization() float64 {
	return lgs.state.Utilization(lgs.MostUtilizedEdge())
}

// SelectEdge selects edge using roulette wheel selection accordingly to random
// number r in [0, 1). For more information about how edges are selected, refer
// to parameter [Config.Alpha].
func (lgs *LinkGuidedSolver) SelectEdge(r float64) int {
	if r < 0 || 1 <= r {
		log.Fatalf("r must be a random number in [0, 1), got: %f", r)
	}
	return lgs.edgeWheel.Get(r * lgs.edgeWheel.TotalWeight())
}

// SelectDemand returns the demand that sends the most traffic on the given
// edge. It returns -1 if no demand sends traffic on the edge.
func (lgs *LinkGuidedSolver) SelectDemand(edge int) int {
	bestLoad := int64(0)
	bestDemand := -1
	for d, l := range lgs.demandsByEdge[edge] {
		switch {
		case l == bestLoad && d < bestDemand:
			bestDemand = d
		case bestLoad < l:
			bestLoad = l
			bestDemand = d
		}
	}
	return bestDemand
}

// Search searches for a move that reduces the load of edge by changing the
// demand's path. The second returned value is a bool that indicates whether a
// valid move was found. Moves that increase the utilization of an edge above
// maxUtil are not considered valid. If several moves are possible, the one that
// reduces the edge's load the most is returned.
func (lgs *LinkGuidedSolver) Search(edge int, demand int, maxUtil float64) (srte.Move, bool) {
	return lgs.state.Search(edge, demand, maxUtil)
}

// ApplyMove applies the move if possible. It returns true if the move was
// applied, false otherwise.
func (lgs *LinkGuidedSolver) ApplyMove(move srte.Move) bool {
	// Apply the move but do not persist the changes yet (see below).
	if applied := lgs.state.ApplyMove(move, false); !applied {
		return false
	}

	// Update structures for fast selection by iterating on the edges that
	// were impacted by the move.
	for _, lc := range lgs.state.Changes() {
		util := lgs.state.Utilization(lc.Edge)
		lgs.edgeWheel.SetWeight(lc.Edge, math.Pow(util, lgs.cfg.Alpha))
		lgs.edgesByUtil.Put(lc.Edge, -util) // non-decreasing order

		// Efficiently maintain the list of demands on each edge by
		// comparing the load change and how much traffic the demand was
		// sending on the edge prior to the change.
		prev := lgs.demandsByEdge[lc.Edge][move.Demand]
		switch delta := lgs.state.Load(lc.Edge) - lc.PreviousLoad; {
		case prev == 0: // the demand was not sending traffic before
			lgs.demandsByEdge[lc.Edge][move.Demand] = delta
		case prev == -delta: // all the traffic has been removed
			delete(lgs.demandsByEdge[lc.Edge], move.Demand)
		default: // the traffic has changed but it is non-null
			lgs.demandsByEdge[lc.Edge][move.Demand] += delta
		}
	}

	// Persist the changes now that the structures have been updated.
	lgs.state.PersistChanges()
	return true
}
