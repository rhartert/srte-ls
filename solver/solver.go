package solver

import (
	"log"
	"math"

	"github.com/rhartert/srte-ls/srte"
	"github.com/rhartert/yagh"
)

type Config struct {
	Alpha float64
}

type LinkGuidedSolver struct {
	state *srte.SRTE
	cfg   Config

	edgeWheel     *srte.SumTree
	edgesByUtil   *yagh.IntMap[float64]
	demandsByEdge []map[int]int64
}

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

func (lgs *LinkGuidedSolver) MostUtilizedEdge() int {
	entry, _ := lgs.edgesByUtil.Min()
	return entry.Elem
}

func (lgs *LinkGuidedSolver) MaxUtilization() float64 {
	return lgs.state.Utilization(lgs.MostUtilizedEdge())
}

func (lgs *LinkGuidedSolver) SelectEdge(r float64) int {
	if r < 0 || 1 <= r {
		log.Fatalf("r must be a random number in [0, 1), got: %f", r)
	}
	return lgs.edgeWheel.Get(r * lgs.edgeWheel.TotalWeight())
}

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

func (lgs *LinkGuidedSolver) Search(edge int, demand int, maxutil float64) (srte.Move, bool) {
	return lgs.state.Search(edge, demand, maxutil)
}

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
