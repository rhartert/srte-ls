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

type moveType int8

const (
	moveUnknown moveType = iota
	moveClear
	moveRemove
	moveUpdate
	moveInsert
)

type Move struct {
	moveType moveType
	position int
	node     int
	demand   int
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

func (srte *SRTE) ApplyMove(m Move, persist bool) {
	movedApplied := false // whether the move was applied or not

	srte.state.UndoChanges()
	switch m.moveType {
	case moveClear:
		movedApplied = srte.Clear(m.demand)
	case moveRemove:
		movedApplied = srte.Remove(m.demand, m.position)
	case moveUpdate:
		movedApplied = srte.Update(m.demand, m.position, m.node)
	case moveInsert:
		movedApplied = srte.Insert(m.demand, m.position, m.node)
	default:
		log.Fatalf("Cannot apply unknown move: %+v", m)
	}

	if !movedApplied {
		return
	}

	switch m.moveType {
	case moveClear:
		srte.PathVar[m.demand].Clear()
	case moveRemove:
		srte.PathVar[m.demand].Remove(m.position)
	case moveUpdate:
		srte.PathVar[m.demand].Update(m.position, m.node)
	case moveInsert:
		srte.PathVar[m.demand].Insert(m.position, m.node)
	}

	if persist {
		srte.PersistChanges()
	}
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

	return Move{moveType: moveClear, demand: demand}, true
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
				moveType: moveRemove,
				demand:   demand,
				position: p,
			}
			edgeLoad = l
		}
	}

	srte.state.UndoChanges()
	return bestMove, bestMove.moveType != moveUnknown
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
					moveType: moveUpdate,
					demand:   demand,
					position: p,
					node:     n,
				}
				edgeLoad = l
			}
		}
	}

	return bestMove, bestMove.moveType != moveUnknown
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
					moveType: moveInsert,
					demand:   demand,
					position: p,
					node:     n,
				}
				edgeLoad = l
			}
		}
	}

	srte.state.UndoChanges()
	return bestMove, bestMove.moveType != moveUnknown
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
