package ring

import (
	"fmt"
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
	// buf holds the backing slice. Its capacity
	// is always a power of two or zero.
	//
	// The length of the buffer is used unconventionally.
	// It is used to hold the start of the data.
	//
	// So when the data is contiguous, it's held in
	// 	buf[len(buf):len(buf)+len]
	// When the data overlaps the end of the buffer,
	// it's held in
	//	buf[len(buf):cap(buf)], buf[:len - (cap(buf)-len(buf))]
	buf []T

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
		s0, s1 := b.slices()
		for _, x := range s0 {
			if !yield(x) {
				return
			}
		}
		for _, x := range s1 {
			if !yield(x) {
				return
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
	buf, i0, _ := b.get()
	return buf[i0]
}

// PushStart pushes an element to the start of the buffer.
func (b *Buffer[T]) PushStart(x T) {
	b.ensureCap(b.Len() + 1)

	buf, i0, _ := b.get()
	i0 = b.mod(i0 + b.Cap() - 1)
	buf[i0] = x
	b.buf = b.buf[:i0]
	b.len++
}

// PushEnd adds an element to the end of the buffer.
func (b *Buffer[T]) PushEnd(x T) {
	b.ensureCap(b.Len() + 1)
	buf, _, i1 := b.get()
	buf[i1] = x
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
	buf, _, i1 := b.get()

	if i1+len(src) <= len(buf) {
		copy(buf[i1:], src)
	} else {
		n := copy(buf[i1:], src)
		copy(buf, src[n:])
	}
	b.len += len(src)
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
	buf, i0, _ := b.get()
	if i0-len(src) >= 0 {
		// It fits in the space before the start.
		copy(buf[i0-len(src):i0], src)
	} else {
		n := copy(buf[:i0], src)
		copy(buf[len(buf)-(len(src)-n):], src[n:])
	}
	i0 = b.mod(i0 + len(buf) - len(src))
	b.buf = b.buf[:i0]
	b.len += len(src)
}

// DiscardFromStart discards min(b.Len(), n) elements from
// the start of the buffer and returns the number actually
// discarded
func (b *Buffer[T]) DiscardFromStart(n int) int {
	n = min(b.Len(), n)
	if n == 0 {
		return 0
	}
	buf, i0, _ := b.get()
	if i0+n < len(buf) {
		// All elements being discarded are in the
		// start segment of buf.
		clear(buf[i0:b.mod(i0+n)])
	} else {
		clear(b.buf[i0:])
		clear(b.buf[:n-(len(buf)-i0)])
	}
	i0 = b.mod(i0 + n)
	b.buf = b.buf[:i0]
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
	buf, _, i1 := b.get()
	if i1-n >= 0 {
		// All the elements being discarded are in
		// the end segment of b.buf.
		clear(buf[i1-n:i1])
	} else {
		clear(buf[:i1])
		clear(buf[len(buf)-(n-i1):])
	}
	b.len -= n
	return n
}

// Copy copies min(b.Len(), len(dst)) values into dst
// from index i in the buffer onwards. It does not affect
// the size of the buffer. It returns the number of elements
// actually copied. It panics if i is out of range.
func (b *Buffer[T]) Copy(dst []T, i int) int {
	if i < 0 || i > b.Len() {
		panic("Copy with out of range index")
	}
	s0, s1 := b.slices()
	n := 0
	if i < len(s0) {
		// Start from within s0
		n = copy(dst, s0[i:])
		if n == len(s0[i:]) {
			// We copied all of s0 from index i onwards, now copy s1
			n += copy(dst[n:], s1)
		}
	} else {
		// Start from within s1
		n = copy(dst, s1[i-len(s0):])
	}
	return n
}

// PeekEnd returns the element at the end of the buffer
// without consuming it. It's equivalent to b.Get(b.Len()-1).
func (b *Buffer[T]) PeekEnd() T {
	if b.Len() == 0 {
		panic("PeekEnd called on empty buffer")
	}
	buf, _, i1 := b.get()
	return buf[b.mod(i1-1)]
}

// Len returns the number of elements in the buffer.
func (b *Buffer[T]) Len() int {
	return b.len
}

// Cap returns the capacity of the underlying buffer.
func (b *Buffer[T]) Cap() int {
	return cap(b.buf)
}

// SetCap sets the capacity of the underlying slice
// to at least max(n, b.Len()). This can be used
// to shrink the capacity an over-large buffer.
//
// Note: the resulting capacity can still be as much
// as b.Len() * 2.
func (b *Buffer[T]) SetCap(n int) {
	b.resize(max(n, b.Len()))
}

// Get returns the i'th element in the buffer; the start element
// is at index zero; the end is at b.Len() - 1.
// It panics if i is out of range.
func (b *Buffer[T]) Get(i int) T {
	if i < 0 || i >= b.Len() {
		panic("ring.Buffer.Get called with index out of range")
	}
	buf, i0, _ := b.get()
	return buf[b.mod(i0+i)]
}

// PopStart removes and returns the element from the start of the buffer. If the
// buffer is empty, the call will panic.
func (b *Buffer[T]) PopStart() T {
	if b.Len() <= 0 {
		panic("ring.Buffer.PopStart called on empty buffer")
	}
	buf, i0, _ := b.get()
	x := buf[i0]
	buf[i0] = *new(T)
	i0 = b.mod(i0 + 1)
	b.buf = b.buf[:i0]
	b.len--
	return x
}

// PopStart removes and returns the element from the end of the buffer. If the
// buffer is empty, the call will panic.
func (b *Buffer[T]) PopEnd() T {
	if b.Len() <= 0 {
		panic("ring.Buffer.PopEnd called on empty buffer")
	}
	buf, _, i1 := b.get()
	i1 = b.mod(i1 - 1)
	x := buf[i1]
	buf[i1] = *new(T)
	b.len--
	return x
}

// resizes the buffer if needed to ensure that the capacity is at least n.
func (b *Buffer[T]) ensureCap(n int) {
	if n <= cap(b.buf) {
		return
	}
	b.resize(n)
}

func (b *Buffer[T]) resize(minCap int) {
	newCap := 1 << bits.Len(uint(minCap-1))
	if newCap == b.Cap() {
		return
	}
	buf, i0, i1 := b.get()
	buf1 := make([]T, newCap)
	if i0 < i1 {
		copy(buf1, buf[i0:i1])
	} else {
		n := copy(buf1, buf[i0:])
		copy(buf1[n:], buf[:i1])
	}
	b.buf = buf1[:0]
}

// get returns the full buffer and the indexes into that
// of the start and just after the end elements respectively.
// When i1<i0, the elements are stored at
// buf[i0:], buf[:i1]
func (b *Buffer[T]) get() ([]T, int, int) {
	return b.buf[:cap(b.buf)], len(b.buf), b.mod(len(b.buf) + b.len)
}

func (b *Buffer[T]) slices() ([]T, []T) {
	data, i0, i1 := b.get()
	if i1 >= i0 {
		return data[i0:i1:i1], nil
	}
	return data[i0:], data[:i1]
}

// mod returns x modulo the buffer capacity.
// It relies on the fact that the buffer capacity is
// always a power of 2.
func (b *Buffer[T]) mod(x int) int {
	return x & (cap(b.buf) - 1)
}

func (b *Buffer[T]) String() string {
	buf, i0, i1 := b.get()
	return fmt.Sprintf("{%#v, i0=%d, i1=%d}", buf, i0, i1)
}
