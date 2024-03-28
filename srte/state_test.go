package srte

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestState_Load(t *testing.T) {
	state := NewNetworkState(3)
	want := int64(100)
	state.loads[1] = want

	got := state.Load(1)

	if got != want {
		t.Errorf("Load(): want %d, got %d", want, got)
	}
}

func TestState_AddLoad_oneAdd(t *testing.T) {
	wantChanges := []LoadChange{{0, 0}}
	wantLoads := []int64{10, 0, 0}
	state := NewNetworkState(3)

	state.AddLoad(0, 10)
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_AddLoad_twoAdds(t *testing.T) {
	wantChanges := []LoadChange{{1, 10}}
	wantLoads := []int64{0, 30, 0}
	state := NewNetworkState(3)
	state.loads[1] = 10

	state.AddLoad(1, 10)
	state.AddLoad(1, 10)
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_RemoveLoad_oneRemove(t *testing.T) {
	wantChanges := []LoadChange{{0, 10}}
	wantLoads := []int64{0, 0, 0}
	state := NewNetworkState(3)
	state.loads[0] = 10

	state.RemoveLoad(0, 10)
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_RemoveLoad_twoRemoves(t *testing.T) {
	wantChanges := []LoadChange{{1, 30}}
	wantLoads := []int64{0, 10, 0}
	state := NewNetworkState(3)
	state.loads[1] = 30

	state.RemoveLoad(1, 10)
	state.RemoveLoad(1, 10)
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_PersistChanges(t *testing.T) {
	wantLoads := []int64{0, 10, 20}
	wantChanges := []LoadChange{}
	state := NewNetworkState(3)

	state.AddLoad(1, 10)
	state.AddLoad(2, 10)
	state.AddLoad(2, 10)
	state.PersistChanges()
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_UndoChanges(t *testing.T) {
	wantLoads := []int64{0, 10, 20}
	wantChanges := []LoadChange{}
	state := NewNetworkState(3)
	state.loads[1] = 10
	state.loads[2] = 20

	state.AddLoad(1, 100)
	state.RemoveLoad(2, 10)
	state.UndoChanges()
	gotChanges := state.Changes()

	for e, want := range wantLoads {
		if got := state.Load(e); got != want {
			t.Errorf("Load(%d): want %d, got %d", e, want, got)
		}
	}
	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Changes(): mismatch (-want +got):\n%s", diff)
	}
}

func TestState_Changes(t *testing.T) {
	want := []LoadChange{{1, 0}, {3, 0}}
	state := NewNetworkState(5)

	state.AddLoad(1, 50)
	state.RemoveLoad(3, 100)
	got := state.Changes()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ChangedEdges(): mismatch (-want +got):\n%s", diff)
	}
}
