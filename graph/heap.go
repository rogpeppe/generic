package graph

func NewHeap[E any](items []E, less func(E, E) bool, setIndex func(e *E, i int)) *Heap[E] {
	h := &Heap[E]{
		Items:    items,
		less:     less,
		setIndex: setIndex,
	}
	n := len(h.Items)
	for i := n/2 - 1; i >= 0; i-- {
		h.down(i, n)
	}
	return h
}

type Heap[E any] struct {
	Items    []E
	less     func(E, E) bool
	setIndex func(*E, int)
}

func (h *Heap[E]) Push(x E) {
	h.Items = append(h.Items, x)
	h.up(len(h.Items) - 1)
}

func (h *Heap[E]) Pop() E {
	n := len(h.Items) - 1
	h.swap(0, n)
	h.down(0, n)
	return h.pop()
}

func (h *Heap[E]) Fix(i int) {
	if !h.down(i, len(h.Items)) {
		h.up(i)
	}
}

func (h *Heap[E]) swap(i, j int) {
	h.Items[i], h.Items[j] = h.Items[j], h.Items[i]
	if h.setIndex != nil {
		h.setIndex(&h.Items[i], i)
		h.setIndex(&h.Items[j], j)
	}
}

func (h *Heap[E]) Remove(i int) E {
	n := len(h.Items) - 1
	if n != i {
		h.swap(i, n)
		if !h.down(i, n) {
			h.up(i)
		}
	}
	return h.pop()
}

func (h *Heap[E]) pop() E {
	n := len(h.Items) - 1
	x := h.Items[n]
	h.Items = h.Items[0:n]
	return x
}

func (h *Heap[E]) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !h.less(h.Items[j], h.Items[i]) {
			break
		}
		h.swap(i, j)
		j = i
	}
}

func (h *Heap[E]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.less(h.Items[j2], h.Items[j1]) {
			j = j2 // = 2*i + 2  // right child
		}
		if !h.less(h.Items[j], h.Items[i]) {
			break
		}
		h.swap(i, j)
		i = j
	}
	return i > i0
}
