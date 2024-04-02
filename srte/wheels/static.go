package wheels

import "log"

type StaticWheel struct {
	n int
	// sumWeights represents a complete tree with n leaves. The root of the
	// tree is at index 1. The left child of a node at index i is at i*2, and
	// the right child at i*2+1. The weight of a parent is the sum of its
	// children's weights.
	sumWeights []float64
}

func NewStaticWheel(n int) *StaticWheel {
	return &StaticWheel{
		n:          n,
		sumWeights: make([]float64, n*2),
	}
}

func (st *StaticWheel) SetWeight(elem int, weight float64) {
	i := st.n + elem
	st.sumWeights[i] = weight
	for p := i / 2; p > 0; p = p / 2 {
		l := p * 2
		r := l + 1
		st.sumWeights[p] = st.sumWeights[l] + st.sumWeights[r]
	}
}

func (st *StaticWheel) Roll(roll float64) int {
	if roll < 0 || 1 <= roll {
		log.Fatalf("r must be a random number in [0, 1), got: %f", roll)
	}
	if st.sumWeights[1] == 0 {
		return -1
	}

	w := roll * st.sumWeights[1]
	i := 1
	for i < st.n {
		l := i * 2
		r := l + 1
		if w < st.sumWeights[l] {
			i = l
		} else {
			i = r
			w -= st.sumWeights[l]
		}
	}
	return i - st.n
}
