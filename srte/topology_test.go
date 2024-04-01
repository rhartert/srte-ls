package srte

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTopology(t *testing.T) {
	testCases := []struct {
		desc   string
		edges  []Edge
		nNodes int
		want   *Topology
	}{
		{
			desc: "empty digraph",
			want: &Topology{
				Nexts: [][]int{},
				Edges: []Edge{},
			},
		},
		{
			// 0-->1
			desc:   "one edge",
			edges:  []Edge{{0, 1, 0}},
			nNodes: 2,
			want: &Topology{
				Nexts: [][]int{{0}, nil},
				Edges: []Edge{{0, 1, 0}},
			},
		},
		{
			// 0-->1   2-->3
			desc:   "not connected",
			edges:  []Edge{{0, 1, 1}, {2, 3, 1}},
			nNodes: 4,
			want: &Topology{
				Nexts: [][]int{{0}, nil, {1}, nil},
				Edges: []Edge{{0, 1, 1}, {2, 3, 1}},
			},
		},
		{
			// 0<->1<->2
			// ^       ^
			// |       |
			// +-->3<--+
			desc: "strongly connected",
			edges: []Edge{
				{0, 1, 1}, // edge: 0
				{1, 0, 1}, // edge: 1
				{1, 2, 1}, // edge: 2
				{2, 1, 1}, // edge: 3
				{0, 3, 1}, // edge: 4
				{3, 0, 1}, // edge: 5
				{2, 3, 1}, // edge: 6
				{3, 2, 1}, // edge: 7
			},
			nNodes: 4,
			want: &Topology{
				Nexts: [][]int{
					{0, 4},
					{1, 2},
					{3, 6},
					{5, 7},
				},
				Edges: []Edge{
					{0, 1, 1},
					{1, 0, 1},
					{1, 2, 1},
					{2, 1, 1},
					{0, 3, 1},
					{3, 0, 1},
					{2, 3, 1},
					{3, 2, 1},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := NewTopology(tc.edges, tc.nNodes)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("NewTopology(): mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
