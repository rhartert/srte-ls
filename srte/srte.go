package srte

import (
	"fmt"
	"log"
	"math"

	"github.com/rhartert/srte-ls/srte/paths"
)

type SRTEInstance struct {
	Graph          *Topology
	FGraphs        *FGraphs
	MaxPathNodes   int
	Demands        []Demand
	LinkCapacities []int64
}

type Demand struct {
	From      int
	To        int
	Bandwidth int64
}

type MoveType int8

const (
	MoveUnknown MoveType = iota
	MoveClear
	MoveRemove
	MoveUpdate
	MoveInsert
)

type Move struct {
	MoveType MoveType
	Position int
	Node     int
	Demand   int
}

type SRTE struct {
	FGraphs  *FGraphs
	PathVar  []paths.PathVar
	Instance *SRTEInstance
	state    *NetworkState
}

// SplitLoad calculates the portion of the load based on the given ratio and
// converts it to an integer to guarantee numerical stability. SplitLoad is
// used for all splitting operations in this package.
func SplitLoad(load int64, ratio float64) int64 {
	return int64(math.Ceil(float64(load) * ratio))
}

func NewSRTE(instance *SRTEInstance) (*SRTE, error) {
	nEdges := len(instance.Graph.Edges)

	fgraphs, err := NewFGraphs(instance.Graph)
	if err != nil {
		return nil, fmt.Errorf("error building fgraphs: %s", err)
	}

	// Initialize one path variable for each demand.
	demandPaths := make([]paths.PathVar, 0, len(instance.Demands))
	for _, d := range instance.Demands {
		demandPaths = append(demandPaths, paths.New(d.From, d.To, instance.MaxPathNodes-1))
	}

	// Initialize the network state and add the traffic of each demand.
	state := NewNetworkState(nEdges)
	for _, d := range instance.Demands {
		for _, er := range fgraphs.EdgeRatios(d.From, d.To) {
			state.AddLoad(er.Edge, SplitLoad(d.Bandwidth, er.Ratio))
		}
	}
	state.PersistChanges() // mark the initial state

	return &SRTE{
		FGraphs:  fgraphs,
		PathVar:  demandPaths,
		Instance: instance,
		state:    state,
	}, nil
}

func (srte *SRTE) Load(edge int) int64 {
	return srte.state.Load(edge)
}

func (srte *SRTE) Utilization(edge int) float64 {
	return float64(srte.state.Load(edge)) / float64(srte.Instance.LinkCapacities[edge])
}

func (srte *SRTE) ApplyMove(m Move, persist bool) bool {
	movedApplied := false // whether the move was applied or not

	srte.state.UndoChanges()
	switch m.MoveType {
	case MoveClear:
		movedApplied = srte.Clear(m.Demand)
	case MoveRemove:
		movedApplied = srte.Remove(m.Demand, m.Position)
	case MoveUpdate:
		movedApplied = srte.Update(m.Demand, m.Position, m.Node)
	case MoveInsert:
		movedApplied = srte.Insert(m.Demand, m.Position, m.Node)
	default:
		log.Fatalf("Cannot apply unknown move: %+v", m)
	}

	if !movedApplied {
		return false
	}

	switch m.MoveType {
	case MoveClear:
		srte.PathVar[m.Demand].Clear()
	case MoveRemove:
		srte.PathVar[m.Demand].Remove(m.Position)
	case MoveUpdate:
		srte.PathVar[m.Demand].Update(m.Position, m.Node)
	case MoveInsert:
		srte.PathVar[m.Demand].Insert(m.Position, m.Node)
	}

	if persist {
		srte.PersistChanges()
	}
	return true
}

func (srte *SRTE) PersistChanges() {
	srte.state.PersistChanges()
}

func (srte *SRTE) Changes() []LoadChange {
	return srte.state.Changes()
}

func (srte *SRTE) Search(edge int, demand int, maxutil float64) (Move, bool) {
	if move, ok := srte.SearchClear(edge, demand, maxutil); ok {
		return move, true
	}
	if move, ok := srte.SearchRemove(edge, demand, maxutil); ok {
		return move, true
	}
	if move, ok := srte.SearchUpdate(edge, demand, maxutil); ok {
		return move, true
	}
	if move, ok := srte.SearchInsert(edge, demand, maxutil); ok {
		return move, true
	}
	return Move{}, false
}

func (srte *SRTE) SearchClear(edge int, demand int, maxUtil float64) (Move, bool) {
	edgeLoad := srte.state.Load(edge)

	srte.state.UndoChanges()
	ok := !srte.Clear(demand)
	srte.state.UndoChanges()

	if !ok {
		return Move{}, false
	}
	if !srte.checkMaxUtil(maxUtil) {
		return Move{}, false
	}
	if l := srte.state.Load(edge); l >= edgeLoad {
		return Move{}, false
	}

	return Move{MoveType: MoveClear, Demand: demand}, true
}

func (srte *SRTE) SearchRemove(edge int, demand int, maxUtil float64) (Move, bool) {
	edgeLoad := srte.state.Load(edge)
	pathVar := srte.PathVar[demand]

	bestMove := Move{}
	for p := 1; p < pathVar.Length(); p++ {
		srte.state.UndoChanges()
		if !srte.Remove(demand, p) {
			continue
		}
		if !srte.checkMaxUtil(maxUtil) {
			continue
		}
		if l := srte.state.Load(edge); l < edgeLoad {
			bestMove = Move{
				MoveType: MoveRemove,
				Demand:   demand,
				Position: p,
			}
			edgeLoad = l
		}
	}

	srte.state.UndoChanges()
	return bestMove, bestMove.MoveType != MoveUnknown
}

func (srte *SRTE) SearchUpdate(edge int, demand int, maxUtil float64) (Move, bool) {
	nNodes := len(srte.Instance.Graph.Nexts)
	edgeLoad := srte.state.Load(edge)
	pathVar := srte.PathVar[demand]

	bestMove := Move{}
	for p := 1; p < pathVar.Length(); p++ {
		for n := 0; n < nNodes; n++ {
			srte.state.UndoChanges()
			if !srte.Update(demand, p, n) {
				continue
			}
			if !srte.checkMaxUtil(maxUtil) {
				continue
			}
			if l := srte.state.Load(edge); l < edgeLoad {
				bestMove = Move{
					MoveType: MoveUpdate,
					Demand:   demand,
					Position: p,
					Node:     n,
				}
				edgeLoad = l
			}
		}
	}

	return bestMove, bestMove.MoveType != MoveUnknown
}

func (srte *SRTE) SearchInsert(edge int, demand int, maxUtil float64) (Move, bool) {
	nNodes := len(srte.Instance.Graph.Nexts)
	edgeLoad := srte.state.Load(edge)
	pathVar := srte.PathVar[demand]
	bestMove := Move{}

	for p := 1; p <= pathVar.Length(); p++ {
		for n := 0; n < nNodes; n++ {
			srte.state.UndoChanges()
			if !srte.Insert(demand, p, n) {
				continue
			}
			if !srte.checkMaxUtil(maxUtil) {
				continue
			}
			if l := srte.state.Load(edge); l < edgeLoad {
				bestMove = Move{
					MoveType: MoveInsert,
					Demand:   demand,
					Position: p,
					Node:     n,
				}
				edgeLoad = l
			}
		}
	}

	srte.state.UndoChanges()
	return bestMove, bestMove.MoveType != MoveUnknown
}

// checkMaxUtil returns true if the utilization of all the changed edges is
// lower than maxUtil.
func (srte *SRTE) checkMaxUtil(maxUtil float64) bool {
	for _, lc := range srte.state.Changes() {
		if srte.Utilization(lc.Edge) >= maxUtil {
			return false
		}
	}
	return true
}

func (srte *SRTE) Clear(demand int) bool {
	p := srte.PathVar[demand]
	if !p.CanClear() {
		return false
	}

	// Before: from -> ... -> node -> ... -> to
	// After:  from -----------------------> to
	nodes := p.Nodes()
	bw := srte.Instance.Demands[demand].Bandwidth
	for i := 1; i < len(nodes); i++ {
		srte.removeLoad(nodes[i-1], nodes[i], bw)
	}
	srte.addLoad(nodes[0], nodes[len(nodes)-1], bw)

	return true
}

func (srte *SRTE) Remove(demand int, pos int) bool {
	p := srte.PathVar[demand]
	if !p.CanRemove(pos) {
		return false
	}

	// Before: prev -> node -> next
	// After:  prev ---------> next
	prev := p.Node(pos - 1)
	node := p.Node(pos)
	next := p.Node(pos + 1)
	bw := srte.Instance.Demands[demand].Bandwidth
	srte.removeLoad(prev, node, bw)
	srte.removeLoad(node, next, bw)
	srte.addLoad(prev, next, bw)

	return true
}

func (srte *SRTE) Update(demand int, pos int, newNode int) bool {
	p := srte.PathVar[demand]
	if !p.CanUpdate(pos, newNode) {
		return false
	}

	// Before: prev -> oldnode -> next
	// After:  prev -> newNode -> next
	prev := p.Node(pos - 1)
	oldNode := p.Node(pos)
	next := p.Node(pos + 1)
	bw := srte.Instance.Demands[demand].Bandwidth
	srte.removeLoad(prev, oldNode, bw)
	srte.removeLoad(oldNode, next, bw)
	srte.addLoad(prev, newNode, bw)
	srte.addLoad(newNode, next, bw)

	return true
}

func (srte *SRTE) Insert(demand int, pos int, node int) bool {
	p := srte.PathVar[demand]
	if !p.CanInsert(pos, node) {
		return false
	}

	// Before: prev ---------> next
	// After:  prev -> node -> next
	prev := p.Node(pos - 1)
	next := p.Node(pos)
	bw := srte.Instance.Demands[demand].Bandwidth
	srte.removeLoad(prev, next, bw)
	srte.addLoad(prev, node, bw)
	srte.addLoad(node, next, bw)

	return true
}

func (srte *SRTE) removeLoad(from int, to int, bw int64) {
	for _, er := range srte.FGraphs.EdgeRatios(from, to) {
		srte.state.RemoveLoad(er.Edge, SplitLoad(bw, er.Ratio))
	}
}

func (srte *SRTE) addLoad(from int, to int, bw int64) {
	for _, er := range srte.FGraphs.EdgeRatios(from, to) {
		srte.state.AddLoad(er.Edge, SplitLoad(bw, er.Ratio))
	}
}
