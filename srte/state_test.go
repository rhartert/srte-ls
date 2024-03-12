package srte

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestState_Load(t *testing.T) {
	state := NewState(3)
	want := int64(100)
	state.loads[1] = want

	got := state.Load(1)

	if got != want {
		t.Errorf("Load(): want %d, got %d", want, got)
	}
}

func TestState_AddLoad_oneAdd(t *testing.T) {
	state := NewState(3)

	state.AddLoad(0, 10)

	if got := state.loads[0]; got != 10 {
		t.Errorf("Load of edge 0 should be 10, got %d", got)
	}
	if !state.savedEdges.Contains(0) {
		t.Errorf("Edge 0 should be saved")
	}
	if got := state.savedLoads[0]; got != 0 {
		t.Errorf("Saved load of edge 0 should be 0, got %d", got)
	}
}

func TestState_AddLoad_twoAdds(t *testing.T) {
	state := NewState(3)
	state.loads[1] = 10

	state.AddLoad(1, 10)
	state.AddLoad(1, 10)

	if got := state.loads[1]; got != 30 {
		t.Errorf("Load of edge 1 should be 30, got %d", got)
	}
	if !state.savedEdges.Contains(1) {
		t.Errorf("Edge 1 should be saved")
	}
	if got := state.savedLoads[1]; got != 10 {
		t.Errorf("Saved load of edge 1 should be 10, got %d", got)
	}
}

func TestState_RemoveLoad_oneRemove(t *testing.T) {
	state := NewState(3)
	state.loads[0] = 10

	state.RemoveLoad(0, 10)

	if got := state.loads[0]; got != 0 {
		t.Errorf("Load of edge 0 should be 0, got %d", got)
	}
	if !state.savedEdges.Contains(0) {
		t.Errorf("Edge 0 should be saved")
	}
	if got := state.savedLoads[0]; got != 10 {
		t.Errorf("Saved load of edge 0 should be 10, got %d", got)
	}
}

func TestState_RemoveLoad_twoRemoves(t *testing.T) {
	state := NewState(3)
	state.loads[1] = 30

	state.RemoveLoad(1, 10)
	state.RemoveLoad(1, 10)

	if got := state.loads[1]; got != 10 {
		t.Errorf("Load of edge 1 should be 10, got %d", got)
	}
	if !state.savedEdges.Contains(1) {
		t.Errorf("Edge 1 should be saved")
	}
	if got := state.savedLoads[1]; got != 30 {
		t.Errorf("Saved load of edge 1 should be 30, got %d", got)
	}
}

func TestState_PersistChanges(t *testing.T) {
	state := NewState(3)

	state.AddLoad(1, 10)
	state.AddLoad(2, 10)
	state.AddLoad(2, 10)
	state.PersistChanges()

	if state.savedEdges.Contains(1) {
		t.Errorf("Edge 1 should not be saved")
	}
	if got := state.loads[1]; got != 10 {
		t.Errorf("Load of edge 1 should be 10, got %d", got)
	}
	if state.savedEdges.Contains(1) {
		t.Errorf("Edge 2 should not be saved")
	}
	if got := state.loads[2]; got != 20 {
		t.Errorf("Load of edge 2 should be 20, got %d", got)
	}
}

func TestState_UndoChanges(t *testing.T) {
	state := NewState(3)
	state.loads[1] = 10
	state.loads[2] = 20

	state.AddLoad(1, 100)
	state.RemoveLoad(2, 10)
	state.UndoChanges()

	if state.savedEdges.Contains(1) {
		t.Errorf("Edge 1 should not be saved")
	}
	if got := state.loads[1]; got != 10 {
		t.Errorf("Load of edge 1 should be 10, got %d", got)
	}
	if state.savedEdges.Contains(1) {
		t.Errorf("Edge 2 should not be saved")
	}
	if got := state.loads[2]; got != 20 {
		t.Errorf("Load of edge 2 should be 20, got %d", got)
	}
}

func TestState_ChangedEdges(t *testing.T) {
	state := NewState(5)
	want := []int{1, 3}

	state.AddLoad(1, 50)
	state.RemoveLoad(3, 100)
	got := state.ChangedEdges()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ChangedEdges(): mismatch (-want +got):\n%s", diff)
	}
}
