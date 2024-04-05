// Package paths provides efficient functions for representing and manipulating
// Segment Routing paths within a network.
package paths

import (
	"fmt"
	"strings"
)

// PathVar represents a valid path between two nodes in a network.
//
// A PathVar is a slice of node IDs that defines a valid path. It respects the
// following invariants:
//
//   - Minimum length: 2 (source and destination nodes)
//   - Source node: First element in the slice
//   - Destination node: Last element in the slice
//   - Unique nodes: No consecutive nodes can be the same
//
// All operations on PathVar guarantee that these invariants are maintained.
type PathVar struct {
	path   []int
	length int
}

// New instantiates and returns a new PathVar.
func New(from int, to int, maxNodes int) *PathVar {
	p := &PathVar{path: make([]int, maxNodes)}
	p.path[0] = from
	p.path[1] = to
	p.length = 2
	return p
}

// Length returns the length of the path in terms of nodes.
func (p *PathVar) Length() int {
	return p.length
}

// Node returns the node at position pos starting from 0 (the source) and
// ending at Length()-1 (the destination).
func (p *PathVar) Node(pos int) int {
	return p.path[pos]
}

// Nodes returns the sequence of nodes in the path (including the path's source
// and destination).
//
// Important: the slice is a view on one of the path's internal structure and
// should only be used in read-only operations. Modifying the slice will most
// likely results in incorrect behavior.
func (p *PathVar) Nodes() []int {
	return p.path[:p.length]
}

// CanClear returns true if the Clear operation can be performed.
func (p *PathVar) CanClear() bool {
	return p.length > 2
}

// Clear attempts to remove all the intermediate nodes between the path's source
// and destination. It returns true if the operation succeeded or false if the
// operation would violate one of the path invariants.
func (p *PathVar) Clear() bool {
	if !p.CanClear() {
		return false
	}
	p.path[1] = p.path[p.length-1]
	p.length = 2
	return true
}

// CanRemove returns true if the Remove operation can be performed on the node
// at the specified position.
func (p *PathVar) CanRemove(pos int) bool {
	return 0 < pos && pos < p.length-1
}

// Remove removes the node at the specified position from the path. It returns
// true if the operation succeeded or false if the operation would violate one
// of the path invariants.
//
// Note that this function might remove more than one node to guarantee that
// the PathVar invariants are maintained. For example, removing node 3 in path
// 1 -> 2 -> 3 -> 2 -> 4  will results in path 1 -> 2 -> 4 to guarantee that no
// consecutive nodes in the path are the same.
func (p *PathVar) Remove(pos int) bool {
	if !p.CanRemove(pos) {
		return false
	}

	offset := 1
	if p.path[pos-1] == p.path[pos+1] {
		offset = 2
	}

	p.length -= offset
	for i := pos; i < p.length; i++ {
		p.path[i] = p.path[i+offset]
	}
	return true
}

// CanUpdate returns true if the Update operation can be performed on the node
// at the specified position.
func (p *PathVar) CanUpdate(pos int, node int) bool {
	if pos <= 0 || p.length-1 <= pos {
		return false
	}
	if p.path[pos-1] == node || p.path[pos+1] == node {
		return false
	}
	if p.path[pos] == node {
		return false
	}
	return true
}

// Update updates the node at the specified position with the new node value.
// It returns true if the operation succeeded or false if the operation would
// violate one of the path invariants.
func (p *PathVar) Update(pos int, node int) bool {
	if !p.CanUpdate(pos, node) {
		return false
	}
	p.path[pos] = node
	return true
}

// CanInsert returns true if the Insert operation can be performed at the
// specified position with the given node.
func (p *PathVar) CanInsert(pos int, node int) bool {
	if p.length == len(p.path) {
		return false
	}
	if pos < 1 || p.length <= pos {
		return false
	}
	if p.path[pos-1] == node || p.path[pos] == node {
		return false
	}
	return true
}

// Insert inserts the given node at the position pos in the path. The node
// originally at position pos (and all subsequent nodes) are shifted one
// position to the right to make space for the new node. The function returns
// true if the operation succeeded or false if the operation would violate one
// of the path invariants.
func (p *PathVar) Insert(pos int, node int) bool {
	if !p.CanInsert(pos, node) {
		return false
	}
	for i := p.length; i > pos; i-- {
		p.path[i] = p.path[i-1]
	}
	p.length += 1
	p.path[pos] = node
	return true
}

// String returns a string representation of the path as a sequence of nodes
// separated by " -> ". For example: "0 -> 4 -> 3 -> 1".
func (p *PathVar) String() string {
	sb := strings.Builder{}
	for i := 0; i < p.length-1; i++ {
		sb.WriteString(fmt.Sprintf("%d -> ", p.path[i]))
	}
	sb.WriteString(fmt.Sprintf("%d", p.path[p.length-1]))
	return sb.String()
}
