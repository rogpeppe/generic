package ring

import (
	"iter"
	"math/bits"
)

// Buffer holds a slice-backed ring buffer. Elements
// can be added and removed at both the start and
// the end of the buffer.
//
// Elements are indexed from zero (the start)
// to the end. Pushing elements at the start
// will implicitly reindex all previous elements.
//
// The zero-value is OK to use.
type Buffer[T any] struct {
	// buf holds the backing slice. Its length
	// is always a power of two or zero.
	buf []T

	// i0 and i1 hold the indexes into buf of the
	// start and just after the end elements respectively.
	// When i1<i0, the elements are stored at
	// buf[i0:], buf[:i1]
	i0, i1 int

	// len holds the number of elements in the buffer.
	len int
}

// NewBuffer returns a buffer with at least the specified capacity.
func NewBuffer[T any](minCap int) *Buffer[T] {
	var b Buffer[T]
	b.ensureCap(minCap)
	return &b
}

// All returns an iterator over all the values in the buffer.
func (b *Buffer[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		n := b.Len()
		for i := range n {
			if !yield(b.Get(i)) {
				break
			}
		}
	}
}

// PeekStart returns the element at the start of the buffer
// without consuming it. It's equivalent to b.Get(0),
// and panics if the buffer is empty.
func (b *Buffer[T]) PeekStart() T {
	if b.Len() <= 0 {
		panic("PeekStart called on empty buffer")
	}
	return b.buf[b.i0]
}

// PushStart pushes an element to the start of the buffer.
func (b *Buffer[T]) PushStart(x T) {
	b.ensureCap(b.Len() + 1)
	b.i0 = b.mod(b.i0 + len(b.buf) - 1)
	b.buf[b.i0] = x
	b.len++
}

// PushEnd adds an element to the end of the buffer.
func (b *Buffer[T]) PushEnd(x T) {
	b.ensureCap(b.Len() + 1)
	b.buf[b.i1] = x
	b.i1 = b.mod(b.i1 + 1)
	b.len++
}

// PushSliceEnd pushes all the elements of the
// given slice onto the end of the buffer.
// It's just like:
//
//	for _, x := range src {
//		b.PushEnd(x)
//	}
//
// but more efficient.
func (b *Buffer[T]) PushSliceEnd(src []T) {
	b.ensureCap(b.Len() + len(src))
	if b.i1+len(src) <= len(b.buf) {
		copy(b.buf[b.i1:], src)
		b.i1 += len(src)
	} else {
		n := copy(b.buf[b.i1:], src)
		copy(b.buf, src[n:])
	}
}

// PushSliceStart pushes all the elements of the
// given slice onto the end of the buffer.
// It's just like:
//
//	for i := len(src)-1; i>=0; i-- {
//		b.PushStart(src[i])
//	}
//
// but more efficient.
func (b *Buffer[T]) PushSliceStart(src []T) {
	b.ensureCap(b.Len() + len(src))
	if b.i0-len(src) >= 0 {
		// It fits in the space before the start.
		copy(b.buf[:b.i0-len(src)], src)
		b.i0 = b.mod(b.i0 + len(b.buf) - len(src))
	} else {
		n := copy(b.buf[:b.i0], src)
		copy(b.buf[len(b.buf)-n:], src)
	}
}

// DiscardFromStart discards min(b.Len(), n) elements from
// the start of the buffer and returns the number actually
// discarded
func (b *Buffer[T]) DiscardFromStart(n int) int {
	n = min(b.Len(), n)
	if n == 0 {
		return 0
	}
	if b.i0+n < len(b.buf) {
		// All elements being discarded are in the
		// start segment of b.buf.
		clear(b.buf[b.i0:b.mod(b.i0+n)])
	} else {
		clear(b.buf[b.i0:])
		clear(b.buf[:n-(len(b.buf)-b.i0)])
	}
	b.i0 = b.mod(b.i0 + n)
	b.len -= n
	return n
}

// DiscardFromEnd discards min(b.Len(), n) elements from
// the end of the buffer and returns the number actually
// discarded
func (b *Buffer[T]) DiscardFromEnd(n int) int {
	n = min(b.Len(), n)
	if n == 0 {
		return 0
	}
	if b.i1-n >= 0 {
		// All the elements being discarded are in
		// the end segment of b.buf.
		clear(b.buf[b.i1-n:])
	} else {
		clear(b.buf[:b.i1])
		clear(b.buf[len(b.buf)-(n-b.i1):])
	}
	b.i1 = b.mod(b.i1 + len(b.buf) - n)
	b.len -= n
	return n
}

// Copy copies min(b.Len(), len(dst)) values into dst
// from index i in the buffer onwards. It does not affect
// the size of the buffer. It returns the number of elements
// actually copied. It panics if i is out of range.
func (b *Buffer[T]) Copy(dst []T, i int) int {
	if i < 0 || i > b.Len() {
		panic("Copy with out of range from value")
	}
	n := min(b.Len()-i, len(dst))
	if n == 0 {
		return 0
	}
	dst = dst[:n]
	if b.i0+i <= len(b.buf) {
		// Easy case: it's all in contiguous elements.
		copy(dst, b.buf[b.mod(b.i0+i):])
	} else {
		nc := copy(dst, b.buf[b.i0+i:min(len(b.buf), n)])
		if nc == n {
			return n
		}
		copy(dst[nc:], b.buf[:b.mod(b.i0+i+nc)])
	}
	return n
}

// PeekEnd returns the element at the end of the buffer
// without consuming it. It's equivalent to b.Get(b.Len()-1).
func (b *Buffer[T]) PeekEnd() T {
	if b.Len() == 0 {
		panic("PeekEnd called on empty buffer")
	}
	return b.buf[b.mod(b.i1-1)]
}

// Len returns the number of elements in the buffer.
func (b *Buffer[T]) Len() int {
	return b.len
}

// Cap returns the capacity of the underlying buffer.
func (b *Buffer[T]) Cap() int {
	return len(b.buf)
}

// SetCap sets the capacity of the underlying slice
// to at least max(n, b.Len()). This can be used
// to shrink the capacity an over-large buffer.
//
// Note: the resulting capacity can still be as much
// as b.Len() * 2.
func (b *Buffer[T]) SetCap(n int) {
	b.resize(n)
}

// Get returns the i'th element in the buffer; the start element
// is at index zero; the end is at b.Len() - 1.
// It panics if i is out of range.
func (b *Buffer[T]) Get(i int) T {
	if i < 0 || i >= b.Len() {
		panic("ring.Buffer.Get called with index out of range")
	}
	return b.buf[b.mod(b.i0+i)]
}

// PopStart removes and returns the element from the start of the buffer. If the
// buffer is empty, the call will panic.
func (b *Buffer[T]) PopStart() T {
	if b.Len() <= 0 {
		panic("ring.Buffer.PopStart called on empty buffer")
	}
	x := b.buf[b.i0]
	b.buf[b.i0] = *new(T)
	b.i0 = b.mod(b.i0 + 1)
	b.len--
	return x
}

// PopStart removes and returns the element from the end of the buffer. If the
// buffer is empty, the call will panic.
func (b *Buffer[T]) PopEnd() T {
	if b.Len() <= 0 {
		panic("ring.Buffer.PopEnd called on empty buffer")
	}
	x := b.buf[b.mod(b.i1-1)]
	b.buf[b.i1] = *new(T)
	b.i1 = b.mod(b.i1 + len(b.buf) - 1)
	b.len--
	return x
}

// resizes the buffer if needed to ensure that the capacity is at least n.
func (b *Buffer[T]) ensureCap(n int) {
	if n <= len(b.buf) {
		return
	}
	b.resize(n)
}

func (b *Buffer[T]) resize(minCap int) {
	newCap := 1 << bits.Len(uint(minCap-1))
	if newCap == b.Cap() {
		return
	}
	buf1 := make([]T, newCap)
	if b.i0 < b.i1 {
		copy(buf1, b.buf[b.i0:b.i1])
	} else {
		n := copy(buf1, b.buf[b.i0:])
		copy(buf1[n:], b.buf[:b.i1])
	}
	b.i0 = 0
	b.i1 = b.len
	b.buf = buf1
}

// mod returns x modulo the buffer capacity.
// It relies on the fact that the buffer capacity is
// always a power of 2.
func (b *Buffer[T]) mod(x int) int {
	return x & (len(b.buf) - 1)
}
