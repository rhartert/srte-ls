// Package solver contains an implementation of the Link-Guided Search algorithm
// to minimize the maximum utilization of a network.
package solver

import (
	"math"

	"github.com/rhartert/srte-ls/srte"
	"github.com/rhartert/srte-ls/srte/wheels"
	"github.com/rhartert/yagh"
)

type Config struct {
	// Parameter alpha controls the likelihood of selecting an edge where the
	// likelihood P(e) of selecting edge e is determined by its utilization
	// raised to the power of alpha: P(e) = (util[e]^alpha) / Σ(util[ei]^alpha).
	// High values of alpha increase the probability of selecting the most
	// utilized edges. By contrast, small values of alpha flatten the
	// probability distribution. In particular, setting alpha to zero results
	// in random uniform selection.
	Alpha float64

	// WARNING: this parameter is currently ignored by the solver.
	//
	// Parameter beta controls the likelihood of selecting a demand where the
	// likelihood P(d|e) of selecting demand d on edge e is determined by the
	// demand's contribution to the utilization of edge e, raised to the power
	// of beta: P(d|e) = (util[e, d]^beta) / Σ(util[e, di]^alpha). High values
	// of beta increase the probability of selecting the demand with the highest
	// contribution. By contrast, small values of beta flatten the probability
	// distribution. In particular, setting alpha to zero results in random
	// uniform selection.
	Beta float64
}

type LinkGuidedSolver struct {
	state *srte.SRTE
	cfg   Config

	edgeWheel    *wheels.StaticWheel
	edgesByUtil  *yagh.IntMap[float64]
	demandWheels []*wheels.DemandWheel
}

// NewLinkGuidedSolver returns a new instance of a Link-Guided Search solver to
// minimize the maximum utilization of the given SRTE state.
func NewLinkGuidedSolver(state *srte.SRTE, cfg Config) *LinkGuidedSolver {
	nEdges := len(state.Instance.Graph.Edges)

	lgs := &LinkGuidedSolver{
		state:        state,
		cfg:          cfg,
		edgeWheel:    wheels.NewStaticWheel(nEdges),
		edgesByUtil:  yagh.New[float64](nEdges),
		demandWheels: make([]*wheels.DemandWheel, nEdges),
	}

	for e := 0; e < nEdges; e++ {
		lgs.edgeWheel.SetWeight(e, math.Pow(state.Utilization(e), cfg.Alpha))
		lgs.edgesByUtil.Put(e, -state.Utilization(e)) // non-decreasing order
		lgs.demandWheels[e] = wheels.NewDemandWheel(16)
	}

	for i, d := range state.Instance.Demands {
		for _, er := range state.FGraphs.EdgeRatios(d.From, d.To) {
			lgs.demandWheels[er.Edge].Put(i, srte.SplitLoad(d.Bandwidth, er.Ratio))
		}
	}

	return lgs
}

// MostUtilizedEdge returns the ID of the edge with the highest utilization.
// If several edges have the same highest utilization, the one with the smallest
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
// to parameter Alpha in [Config].
func (lgs *LinkGuidedSolver) SelectEdge(r float64) int {
	return lgs.edgeWheel.Roll(r)
}

// SelectDemand selects a demand passing through a given edge using roulette
// wheel selection accordingly to random number r in [0, 1). For more
// information about how demands are selected, refer to parameter Beta in
// [Config].
func (lgs *LinkGuidedSolver) SelectDemand(edge int, r float64) int {
	return lgs.demandWheels[edge].Roll(r)
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

		// Efficiently maintain the list of demands passing through the edge
		// by comparing the load before and after the move. The trick is that
		// the edge load change can only be caused by the demand being moved.
		oldTraffic := lgs.demandWheels[lc.Edge].Get(move.Demand)
		delta := lgs.state.Load(lc.Edge) - lc.PreviousLoad
		newTraffic := oldTraffic + delta
		if newTraffic == 0 {
			lgs.demandWheels[lc.Edge].Remove(move.Demand)
		} else {
			lgs.demandWheels[lc.Edge].Put(move.Demand, newTraffic)
		}
	}

	// Persist the changes now that the structures have been updated.
	lgs.state.PersistChanges()
	return true
}
