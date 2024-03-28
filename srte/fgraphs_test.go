package srte

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFGraphs_EdgeRatios(t *testing.T) {
	want := []EdgeRatio{{1, 0.1}, {2, 0.2}, {3, 0.3}}
	fgs := FGraphs{
		edgesRatios: [][][]EdgeRatio{{{}, want}},
	}

	got := fgs.EdgeRatios(0, 1)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("EdgeRatios(): mismatch (-want +got):\n%s", diff)
	}
}

func TestNew(t *testing.T) {
	testCases := []struct {
		desc    string
		graph   *Digraph
		want    *FGraphs
		wantErr bool
	}{
		{
			desc:  "empty graph",
			graph: NewDigraph(nil, 0),
			want: &FGraphs{
				[][][]EdgeRatio{},
			},
		},
		{
			desc:  "single node",
			graph: NewDigraph(nil, 1),
			want: &FGraphs{
				[][][]EdgeRatio{{nil}},
			},
		},
		{
			// 0-->1
			desc:  "one edge",
			graph: NewDigraph([]Edge{{0, 1, 0}}, 2),
			want: &FGraphs{
				[][][]EdgeRatio{
					{
						nil,      // 0 -> 0
						{{0, 1}}, // 0 -> 1
					},
					{
						{},  // 1 -> 0
						nil, // 1 -> 1
					},
				},
			},
		},
		{
			// 0-->1   2-->3
			desc:  "not connected",
			graph: NewDigraph([]Edge{{0, 1, 1}, {2, 3, 1}}, 4),
			want: &FGraphs{
				[][][]EdgeRatio{
					{
						nil,      // 0 -> 0
						{{0, 1}}, // 0 -> 1
						{},       // 0 -> 2
						{},       // 0 -> 3
					},
					{
						{},  // 1 -> 0
						nil, // 1 -> 1
						{},  // 1 -> 2
						{},  // 1 -> 3
					},
					{
						{},       // 2 -> 0
						{},       // 2 -> 1
						nil,      // 2 -> 2
						{{1, 1}}, // 2 -> 3
					},
					{
						{},  // 3 -> 0
						{},  // 3 -> 1
						{},  // 3 -> 2
						nil, // 3 -> 3
					},
				},
			},
		},
		{
			// 0<--1<--2
			//     ^   ^
			//     |   |
			//     3<--4
			desc: "two paths with bridge",
			graph: NewDigraph([]Edge{
				{1, 0, 1}, // edge: 0
				{2, 1, 1}, // edge: 1
				{3, 1, 1}, // edge: 2
				{4, 2, 1}, // edge: 3
				{4, 3, 1}, // edge: 4
			}, 5),
			want: &FGraphs{
				[][][]EdgeRatio{
					{
						nil,
						{},
						{},
						{},
						{},
					},
					{
						{{0, 1}},
						nil,
						{},
						{},
						{},
					},
					{
						{{0, 1}, {1, 1}},
						{{1, 1}},
						nil,
						{},
						{},
					},
					{
						{{0, 1}, {2, 1}},
						{{2, 1}},
						{},
						nil,
						{},
					},
					{
						{{0, 1}, {1, 0.5}, {2, 0.5}, {3, 0.5}, {4, 0.5}},
						{{1, 0.5}, {2, 0.5}, {3, 0.5}, {4, 0.5}},
						{{3, 1}},
						{{4, 1}},
						nil,
					},
				},
			},
		},
		{
			// 0<->1<->2
			// ^       ^
			// |       |
			// +-->3<--+
			desc: "strongly connected",
			graph: NewDigraph([]Edge{
				{0, 1, 1}, // edge: 0
				{1, 0, 1}, // edge: 1
				{1, 2, 1}, // edge: 2
				{2, 1, 1}, // edge: 3
				{0, 3, 1}, // edge: 4
				{3, 0, 1}, // edge: 5
				{2, 3, 1}, // edge: 6
				{3, 2, 1}, // edge: 7
			}, 4),
			want: &FGraphs{
				[][][]EdgeRatio{
					{
						nil,                                      // 0 -> 0
						{{0, 1}},                                 // 0 -> 1
						{{0, 0.5}, {2, 0.5}, {4, 0.5}, {7, 0.5}}, // 0 -> 2
						{{4, 1}},                                 // 0 -> 3
					},
					{
						{{1, 1}},                                 // 1 -> 0
						nil,                                      // 1 -> 1
						{{2, 1}},                                 // 1 -> 2
						{{1, 0.5}, {2, 0.5}, {4, 0.5}, {6, 0.5}}, // 1 -> 3
					},
					{
						{{1, 0.5}, {3, 0.5}, {5, 0.5}, {6, 0.5}}, // 2 -> 0
						{{3, 1}},                                 // 2 -> 1
						nil,                                      // 2 -> 2
						{{6, 1}},                                 // 2 -> 3
					},
					{
						{{5, 1}},                                 // 3 -> 0
						{{0, 0.5}, {3, 0.5}, {5, 0.5}, {7, 0.5}}, // 3 -> 1
						{{7, 1}},                                 // 3 -> 2
						nil,                                      // 3 -> 3
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got, gotErr := NewFGraphs(tc.graph)

			if tc.wantErr && gotErr == nil {
				t.Errorf("New(): want error, got nil")
			}
			if !tc.wantErr && gotErr != nil {
				t.Errorf("New(): want no error, got %s", gotErr)
			}
			if diff := cmp.Diff(tc.want.edgesRatios, got.edgesRatios); diff != "" {
				t.Errorf("New(): mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNew_shortestPathDAG(t *testing.T) {
	testCases := []struct {
		desc    string
		graph   *Digraph
		src     int
		want    [][]int
		wantErr bool
	}{
		{
			desc:    "nil graph",
			wantErr: true,
		},
		{
			desc:    "empty graph",
			graph:   NewDigraph(nil, 0),
			wantErr: true,
		},
		{
			desc:  "single node (no edge)",
			graph: NewDigraph(nil, 1),
			want:  [][]int{nil},
		},
		{
			// 0-->1
			desc:  "one edge",
			graph: NewDigraph([]Edge{{0, 1, 0}}, 2),
			want:  [][]int{nil, {0}},
		},
		{
			// 0-->1   2-->3
			desc:  "not connected",
			graph: NewDigraph([]Edge{{0, 1, 1}, {2, 3, 1}}, 4),
			want:  [][]int{nil, {0}, nil, nil},
		},
		{
			// 0-->1-->2
			//  \      ^
			//   \     |
			//    +----+
			desc: "one shortest path (A)",
			graph: NewDigraph([]Edge{
				{0, 1, 1},
				{1, 2, 1},
				{0, 2, 3},
			}, 3),
			want: [][]int{nil, {0}, {1}},
		},
		{
			// 0-->1-->2
			//  \      ^
			//   \     |
			//    +----+
			desc: "one shortest path (B)",
			graph: NewDigraph([]Edge{
				{0, 1, 1},
				{1, 2, 1},
				{0, 2, 1},
			}, 3),
			want: [][]int{nil, {0}, {2}},
		},
		{
			// 0-->1-->2
			//  \      ^
			//   \     |
			//    +----+
			desc: "two shortest paths",
			graph: NewDigraph([]Edge{
				{0, 1, 1},
				{1, 2, 1},
				{0, 2, 2},
			}, 3),
			want: [][]int{nil, {0}, {2, 1}},
		},
		{
			// 0-->1-->2-->3
			// |   ^       ^
			// |   |       |
			// +-->4------>5
			desc: "three shortest paths",
			graph: NewDigraph([]Edge{
				{0, 1, 2}, // edge: 0
				{1, 2, 2}, // edge: 1
				{2, 3, 1}, // edge: 2
				{0, 4, 1}, // edge: 3
				{4, 1, 1}, // edge: 4
				{4, 5, 3}, // edge: 5
				{5, 3, 1}, // edge: 6
			}, 6),
			want: [][]int{
				nil,
				{0, 4},
				{1},
				{6, 2},
				{3},
				{5},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got, gotErr := shortestDAG(tc.graph, tc.src)

			if tc.wantErr && gotErr == nil {
				t.Errorf("shortestDAG(): want error, got nil")
			}
			if !tc.wantErr && gotErr != nil {
				t.Errorf("shortestDAG(): want no error, got %s", gotErr)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("shortestDAG(): mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
