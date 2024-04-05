package wheels

import "log"

type DemandWheel struct {
	offset  int
	size    int
	weights []int64
	elems   []int
	hash    map[int]int
}

func NewDemandWheel(initSize int) *DemandWheel {
	offset := nextPower2(initSize)
	return &DemandWheel{
		offset:  offset,
		size:    0,
		weights: make([]int64, offset*2),
		elems:   make([]int, offset*2),
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

func (st *DemandWheel) Put(elem int, weight int64) {
	if i, ok := st.hash[elem]; ok {
		st.update(i, weight)
	} else {
		st.insert(elem, weight)
	}
}

func (st *DemandWheel) update(i int, weight int64) {
	n := st.offset + i
	st.weights[n] = weight
	st.propagate(n)
}

func (st *DemandWheel) insert(elem int, weight int64) {
	if st.offset+st.size == len(st.weights) {
		st.grow()
	}

	n := st.offset + st.size
	st.weights[n] = weight
	st.elems[n] = elem
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
		st.hash[st.elems[lastNode]] = i
		st.propagate(delNode)
	}

	st.weights[lastNode] = 0
	st.propagate(lastNode)
}

func (st *DemandWheel) Get(elem int) int64 {
	if i, ok := st.hash[elem]; ok {
		return st.weights[st.offset+i]
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

	w := int64(float64(st.weights[1]) * roll)
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
	newWeights := make([]int64, newOffset*2)
	newElems := make([]int, newOffset*2)
	copy(newWeights[newOffset:], st.weights[st.offset:])
	copy(newElems[newOffset:], st.elems[st.offset:])
	st.weights = newWeights
	st.elems = newElems
	st.offset = newOffset
	for p := st.offset - 1; p > 0; p-- {
		l := p * 2
		r := l + 1
		st.weights[p] = st.weights[l] + st.weights[r]
	}
}
