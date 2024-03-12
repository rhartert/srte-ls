package srte

import "github.com/rhartert/sparsesets"

// State is a reversible structure which represents the state of the network in
// terms of traffic. This structure keep track of changes applied to its edges
// and can efficiently undo them.
type State struct {
	loads      []int64
	savedLoads []int64
	savedEdges *sparsesets.Set
}

// New initializes and returns a new State.
func NewState(nEdges int) *State {
	return &State{
		loads:      make([]int64, nEdges),
		savedLoads: make([]int64, nEdges),
		savedEdges: sparsesets.New(nEdges),
	}
}

// Load returns the current load on the edge.
func (s State) Load(edge int) int64 {
	return s.loads[edge]
}

// AddLoad adds the load from the edge. The change is registered so that it
// can be undone if needed.
func (s State) AddLoad(edge int, load int64) {
	if !s.savedEdges.Contains(edge) {
		s.savedEdges.Insert(edge)
		s.savedLoads[edge] = s.loads[edge]
	}
	s.loads[edge] += load
}

// RemoveLoad removes the load from the edge. The change is registered so that
// it can be undone if needed.
func (s State) RemoveLoad(edge int, load int64) {
	if !s.savedEdges.Contains(edge) {
		s.savedEdges.Insert(edge)
		s.savedLoads[edge] = s.loads[edge]
	}
	s.loads[edge] -= load
}

// PersistChanges persists all the changes as the "new" state. New changes can
// be accumulated (and undone) from this point.
func (s *State) PersistChanges() {
	s.savedEdges.Clear()
}

// UndoChanges undoes all the changes since the last time PersistChanges was
// called. This operation is done in O(C) where C is the number of edges that
// have been changed.
func (s *State) UndoChanges() {
	for _, e := range s.savedEdges.Content() {
		s.loads[e] = s.savedLoads[e]
	}
	s.savedEdges.Clear()
}

// ChangedEdges returns the edges that have been changed since the last time
// changes were persisted.
//
// Impirtant: the slice is a view on one of the state's internal structure and
// should only be used in read-only operations. Modifying the slice will most
// likely results in incorrect behavior.
func (s *State) ChangedEdges() []int {
	return s.savedEdges.Content()
}
