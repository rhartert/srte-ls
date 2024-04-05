package wheels

import "log"

type DemandWheel struct {
	offset  int
	size    int
	weights []float64
	loads   []int64
	elems   []int
	hash    map[int]int
}

func NewDemandWheel(initSize int) *DemandWheel {
	offset := nextPower2(initSize)
	return &DemandWheel{
		offset:  offset,
		size:    0,
		weights: make([]float64, offset*2),
		elems:   make([]int, offset*2),
		loads:   make([]int64, offset*2),
		hash:    make(map[int]int, initSize),
	}
}

func nextPower2(i int) int {
	i |= i >> 1
	i |= i >> 2
	i |= i >> 4
	i |= i >> 8
	i |= i >> 16
	i |= i >> 32
	return i + 1
}

func (st *DemandWheel) Put(elem int, load int64, weight float64) {
	if _, ok := st.hash[elem]; ok {
		st.update(elem, load, weight)
	} else {
		st.insert(elem, load, weight)
	}
}

func (st *DemandWheel) update(elem int, load int64, weight float64) {
	n := st.offset + st.hash[elem]
	st.loads[n] = load
	st.weights[n] = weight
	st.propagate(n)
}

func (st *DemandWheel) insert(elem int, load int64, weight float64) {
	if st.offset+st.size == len(st.weights) {
		st.grow()
	}

	n := st.offset + st.size
	st.elems[n] = elem
	st.loads[n] = load
	st.weights[n] = weight
	st.hash[elem] = st.size
	st.size++

	st.propagate(n)
}

func (st *DemandWheel) Remove(elem int) {
	i, ok := st.hash[elem]
	if !ok {
		return
	}

	delete(st.hash, elem)

	st.size--
	delNode := st.offset + i
	lastNode := st.offset + st.size

	if delNode != lastNode {
		st.weights[delNode] = st.weights[lastNode]
		st.elems[delNode] = st.elems[lastNode]
		st.loads[delNode] = st.loads[lastNode]
		st.hash[st.elems[lastNode]] = i
		st.propagate(delNode)
	}

	st.weights[lastNode] = 0
	st.propagate(lastNode)
}

func (st *DemandWheel) GetLoad(elem int) int64 {
	if i, ok := st.hash[elem]; ok {
		return st.loads[st.offset+i]
	}
	return 0
}

func (st *DemandWheel) Roll(roll float64) int {
	if roll < 0 || 1 <= roll {
		log.Fatalf("r must be a random number in [0, 1), got: %f", roll)
	}
	if st.size == 0 {
		return -1
	}

	w := float64(st.weights[1]) * roll
	i := 1
	for i < st.offset {
		l := i * 2
		r := l + 1
		if w < st.weights[l] {
			i = l
		} else {
			i = r
			w -= st.weights[l]
		}
	}
	return st.elems[i]
}

func (st *DemandWheel) propagate(i int) {
	for p := i >> 1; p > 0; p = p >> 1 {
		l := p << 1
		r := l + 1
		st.weights[p] = st.weights[l] + st.weights[r]
	}
}

func (st *DemandWheel) grow() {
	newOffset := len(st.weights)
	newWeights := make([]float64, newOffset*2)
	newLoads := make([]int64, newOffset*2)
	newElems := make([]int, newOffset*2)
	copy(newWeights[newOffset:], st.weights[st.offset:])
	copy(newLoads[newOffset:], st.loads[st.offset:])
	copy(newElems[newOffset:], st.elems[st.offset:])
	st.weights = newWeights
	st.loads = newLoads
	st.elems = newElems
	st.offset = newOffset
	for p := st.offset - 1; p > 0; p-- {
		l := p * 2
		r := l + 1
		st.weights[p] = st.weights[l] + st.weights[r]
	}
}
