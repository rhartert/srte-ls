package srte

import "math"

// LoadChange is a pair made of an edge and its load before any change was
// applied to the current state.
type LoadChange struct {
	Edge         int
	PreviousLoad int64
}

// NetworkState is a reversible structure which represents the state of the
// network's load. This structure keeps track of changes applied to its edges
// and can efficiently undo them.
type NetworkState struct {
	loads []int64

	// Stack of changes used to restore the last persisted state.
	changes  []LoadChange
	nChanges int

	// The savedAt slice effectively acts as a slice of booleans to check
	// whether the load of an edge was changed in the current state or not.
	// Precisely, an edge e has been changed if savedAt[e] == timestamp. The
	// use of a logical timestamp (rather than booleans) provides an efficient
	// way to mark all edges as unchanged in O(1) by incrementing the timestamp.
	savedAt   []uint
	timestamp uint
}

// NewNetworkState initializes and returns a new NetworkState.
func NewNetworkState(nEdges int) *NetworkState {
	return &NetworkState{
		loads:     make([]int64, nEdges),
		changes:   make([]LoadChange, nEdges),
		nChanges:  0,
		savedAt:   make([]uint, nEdges),
		timestamp: 1, // must be greater than the zero values in savedAt
	}
}

// Load returns the current load on the edge.
func (s *NetworkState) Load(edge int) int64 {
	return s.loads[edge]
}

// AddLoad adds the load from the edge. The change is registered so that it
// can be undone if needed.
func (s *NetworkState) AddLoad(edge int, load int64) {
	if s.savedAt[edge] != s.timestamp {
		s.changes[s.nChanges] = LoadChange{edge, s.loads[edge]}
		s.nChanges += 1
		s.savedAt[edge] = s.timestamp
	}
	s.loads[edge] += load
}

// RemoveLoad removes the load from the edge. The change is registered so that
// it can be undone if needed.
func (s *NetworkState) RemoveLoad(edge int, load int64) {
	s.AddLoad(edge, -load)
}

// PersistChanges persists all the changes as the "new" state. New changes can
// be accumulated (and undone) from this point.
func (s *NetworkState) PersistChanges() {
	s.nChanges = 0
	s.incrTimestamp()
}

// UndoChanges undoes all the changes since the last time PersistChanges was
// called. This operation is done in O(C) where C is the number of edges that
// have been changed.
func (s *NetworkState) UndoChanges() {
	for s.nChanges > 0 {
		s.nChanges -= 1
		lc := s.changes[s.nChanges]
		s.loads[lc.Edge] = lc.PreviousLoad
	}
	s.incrTimestamp()
}

// Changes returns the edges that have been changed since the last time
// changes were persisted.
//
// Important: the slice is a view on one of the state's internal structure and
// should only be used in read-only operations. Modifying the slice will most
// likely results in incorrect behavior.
func (s *NetworkState) Changes() []LoadChange {
	return s.changes[:s.nChanges]
}

// incrTimestamp safely increments the value of the timestamp by resetting the
// savedAt slice and the timestamp if it overflows.
func (s *NetworkState) incrTimestamp() {
	if s.timestamp != math.MaxUint {
		s.timestamp += 1
		return
	}
	s.timestamp = 1
	for i := range s.savedAt {
		s.savedAt[i] = 0
	}
}
