package srte

type SumTree struct {
	n int
	// sumWeights represents a complete tree with n leaves. The root of the
	// tree is at index 1. The left child of a node at index i is at i*2, and
	// the right child at i*2+1. The weight of a parent is the sume of the
	// weights of its children.
	sumWeights []float64
}

func NewSumTree(n int) *SumTree {
	return &SumTree{
		n:          n,
		sumWeights: make([]float64, n*2),
	}
}

func (st *SumTree) SetWeight(elem int, weight float64) {
	i := st.n + elem
	st.sumWeights[i] = weight
	for p := i / 2; p > 0; p = p / 2 {
		l := p * 2
		r := l + 1
		st.sumWeights[p] = st.sumWeights[l] + st.sumWeights[r]
	}
}

func (st *SumTree) TotalWeight() float64 {
	return st.sumWeights[1]
}

func (st *SumTree) Get(roll float64) int {
	if st.sumWeights[1] == 0 {
		return -1
	}
	i := 1
	for i < st.n {
		l := i * 2
		r := l + 1
		if roll < st.sumWeights[l] {
			i = l
		} else {
			i = r
			roll -= st.sumWeights[l]
		}
	}
	return i - st.n
}
