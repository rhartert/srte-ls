package srte

import (
	"fmt"
	"log"
	"math"

	"github.com/rhartert/sparsesets"
	"github.com/rhartert/srte-ls/srte/paths"
	"github.com/rhartert/yagh"
)

type SRTEInstance struct {
	Graph          *Digraph
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
	FGraphs *FGraphs
	State   *NetworkState
	PathVar []paths.PathVar

	Instance *SRTEInstance

	// Maintain the most loaded edge.
	edgesByUsage *yagh.IntMap[float64]

	// Set of demands sending traffic over each edge.
	//
	// TODO: replace this with a datastructure that will allow a more efficient
	// demand selection (e.g. a linked RB tree).
	edgesToDemand []map[int]bool

	// These two sparse sets are used to efficiently maintain the list of
	// demands passing through each edges. These sets are pre-allocated so that
	// they can be used by several moves.
	edgesBefore *sparsesets.Set
	edgesAfter  *sparsesets.Set
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

	edgesToDemand := make([]map[int]bool, nEdges)
	for e := range edgesToDemand {
		edgesToDemand[e] = map[int]bool{}
	}

	// Initialize the network state and add the traffic of each demand.
	state := NewNetworkState(nEdges)
	for i, d := range instance.Demands {
		for _, er := range fgraphs.EdgeRatios(d.From, d.To) {
			state.AddLoad(er.Edge, load(d.Bandwidth, er.Ratio))
			edgesToDemand[er.Edge][i] = true
		}
	}
	state.PersistChanges() // mark the initial state

	// Order edges by descending usage. This data structure will be maintained
	// through the state of the search to easily access the most loaded edge.
	edgesByUsage := yagh.New[float64](nEdges)
	edgeWheel := NewSumTree(nEdges)
	for e, l := range state.loads {
		util := float64(l) / float64(instance.LinkCapacities[e])
		edgeWheel.SetWeight(e, math.Pow(util, 8))
		edgesByUsage.Put(e, -util)
	}

	return &SRTE{
		FGraphs:       fgraphs,
		State:         state,
		PathVar:       demandPaths,
		Instance:      instance,
		edgesBefore:   sparsesets.New(nEdges),
		edgesAfter:    sparsesets.New(nEdges),
		edgesByUsage:  edgesByUsage,
		edgesToDemand: edgesToDemand,
	}, nil
}

func (srte *SRTE) MostUtilizedEdge() int {
	entry, _ := srte.edgesByUsage.Min()
	return entry.Elem
}

func (srte *SRTE) Utilization(edge int) float64 {
	return float64(srte.State.Load(edge)) / float64(srte.Instance.LinkCapacities[edge])
}

func (srte *SRTE) SelectDemand(edge int) int {
	best := int64(0)
	i := -1
	for k := range srte.edgesToDemand[edge] {
		l := srte.Instance.Demands[k].Bandwidth
		if l > best {
			best = l
			i = k
			continue
		}
		if l == best && k > i {
			i = k
		}
	}
	return i
}

func (srte *SRTE) ApplyMove(m Move, persist bool) {
	movedApplied := false // whether the move was applied or not

	srte.State.UndoChanges()
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

	srte.addDemandEdges(m.demand, srte.edgesBefore)
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
	srte.addDemandEdges(m.demand, srte.edgesAfter)
	srte.updateEdgeDemand(m.demand)

	if persist {
		srte.PersistChanges()
	}
}

func (srte *SRTE) PersistChanges() {
	for _, lc := range srte.State.Changes() {
		cost := -srte.Utilization(lc.Edge) // sort by decreasing utilization
		srte.edgesByUsage.Put(lc.Edge, cost)
	}
	srte.State.PersistChanges()
}

func (srte *SRTE) Changes() []LoadChange {
	return srte.State.Changes()
}

func (srte *SRTE) Search(edge int, demand int, strict bool) (Move, bool) {
	if move, ok := srte.SearchClear(edge, demand, strict); ok {
		return move, true
	}
	if move, ok := srte.SearchRemove(edge, demand, strict); ok {
		return move, true
	}
	if move, ok := srte.SearchUpdate(edge, demand, strict); ok {
		return move, true
	}
	if move, ok := srte.SearchInsert(edge, demand, strict); ok {
		return move, true
	}
	return Move{}, false
}

func (srte *SRTE) SearchClear(edge int, demand int, strict bool) (Move, bool) {
	edgeLoad := srte.State.Load(edge)
	maxUtil := srte.Utilization(srte.MostUtilizedEdge())

	srte.State.UndoChanges()
	ok := !srte.Clear(demand)
	srte.State.UndoChanges()

	if !ok {
		return Move{}, false
	}
	if strict && !srte.checkMaxUtil(maxUtil) {
		return Move{}, false
	}
	if l := srte.State.Load(edge); l >= edgeLoad {
		return Move{}, false
	}

	return Move{moveType: moveClear, demand: demand}, true
}

func (srte *SRTE) SearchRemove(edge int, demand int, strict bool) (Move, bool) {
	edgeLoad := srte.State.Load(edge)
	maxUtil := srte.Utilization(srte.MostUtilizedEdge())
	pathVar := srte.PathVar[demand]

	bestMove := Move{}
	for p := 1; p < pathVar.Length(); p++ {
		srte.State.UndoChanges()
		if !srte.Remove(demand, p) {
			continue
		}
		if strict && !srte.checkMaxUtil(maxUtil) {
			continue
		}
		if l := srte.State.Load(edge); l < edgeLoad {
			bestMove = Move{
				moveType: moveRemove,
				demand:   demand,
				position: p,
			}
			edgeLoad = l
		}
	}

	srte.State.UndoChanges()
	return bestMove, bestMove.moveType != moveUnknown
}

func (srte *SRTE) SearchUpdate(edge int, demand int, strict bool) (Move, bool) {
	nNodes := len(srte.Instance.Graph.Nexts)
	edgeLoad := srte.State.Load(edge)
	maxUtil := srte.Utilization(srte.MostUtilizedEdge())
	pathVar := srte.PathVar[demand]

	bestMove := Move{}
	for p := 1; p < pathVar.Length(); p++ {
		for n := 0; n < nNodes; n++ {
			srte.State.UndoChanges()
			if !srte.Update(demand, p, n) {
				continue
			}
			if strict && !srte.checkMaxUtil(maxUtil) {
				continue
			}
			if l := srte.State.Load(edge); l < edgeLoad {
				bestMove = Move{
					moveType: moveUpdate,
					demand:   demand,
					position: p,
					node:     n,
				}
			}
		}
	}

	return bestMove, bestMove.moveType != moveUnknown
}

func (srte *SRTE) SearchInsert(edge int, demand int, strict bool) (Move, bool) {
	nNodes := len(srte.Instance.Graph.Nexts)
	edgeLoad := srte.State.Load(edge)
	maxUtil := srte.Utilization(srte.MostUtilizedEdge())
	pathVar := srte.PathVar[demand]
	bestMove := Move{}

	for p := 1; p <= pathVar.Length(); p++ {
		for n := 0; n < nNodes; n++ {
			srte.State.UndoChanges()
			if !srte.Insert(demand, p, n) {
				continue
			}
			if strict && !srte.checkMaxUtil(maxUtil) {
				continue
			}
			if l := srte.State.Load(edge); l < edgeLoad {
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

	srte.State.UndoChanges()
	return bestMove, bestMove.moveType != moveUnknown
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

func load(bw int64, ratio float64) int64 {
	return int64(math.Ceil(float64(bw) * ratio))
}

func (srte *SRTE) checkMaxUtil(maxUtil float64) bool {
	for _, lc := range srte.State.Changes() {
		if srte.Utilization(lc.Edge) >= maxUtil {
			return false
		}
	}
	return true
}

func (srte *SRTE) removeLoad(from int, to int, bw int64) {
	for _, er := range srte.FGraphs.EdgeRatios(from, to) {
		srte.State.RemoveLoad(er.Edge, load(bw, er.Ratio))
	}
}

func (srte *SRTE) addLoad(from int, to int, bw int64) {
	for _, er := range srte.FGraphs.EdgeRatios(from, to) {
		srte.State.AddLoad(er.Edge, load(bw, er.Ratio))
	}
}

func (srte *SRTE) addDemandEdges(demand int, set *sparsesets.Set) {
	set.Clear()
	nodes := srte.PathVar[demand].Nodes()
	for i := 1; i < len(nodes); i++ {
		for _, er := range srte.FGraphs.EdgeRatios(nodes[i-1], nodes[i]) {
			set.Insert(er.Edge)
		}
	}
}

func (srte *SRTE) updateEdgeDemand(demand int) {
	for _, e := range srte.edgesBefore.Content() {
		if !srte.edgesAfter.Contains(e) {
			delete(srte.edgesToDemand[e], demand)
		}
	}
	for _, e := range srte.edgesAfter.Content() {
		if !srte.edgesBefore.Contains(e) {
			srte.edgesToDemand[e][demand] = true
		}
	}
	srte.edgesBefore.Clear()
	srte.edgesAfter.Clear()
}
