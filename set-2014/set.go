// The set package provides a generic set interface and
// some implementations of it.
//
// Is this going too far?!
package set

type Equaler[T any] interface {
	Equal(t T) bool
}

// Set defines a mutable set; members can be added
// and removed and the set can be combined with
// itself.
type Set[
	self any,
	elem Equaler[elem],
] interface {
	// New returns a new empty instance of the set.
	New() self

	// Union sets the contents of the receiver to
	// the union of a and b and returns the receiver.
	Union(a, b self) self

	// Intersect sets the contents of the receiver to
	// the union of a and b and returns the receiver.
	Intersect(a, b self) self

	// Iter returns an iterator that visits each member
	// of the set in turn.
	Iter() Iter[elem]

	// Add adds the given elements to the set.
	Add(elem)

	// Add removes the given elements to the set.
	Remove(elem)
}

type Iter[T any] interface {
	Next() bool
	Item() T
}

type BitSet struct {
	bits []uint64
}

type Int int

func (i Int) Equal(j Int) bool {
	return i == j
}

const wbits = 64

func (b *BitSet) New() *BitSet {
	return &BitSet{}
}

func (b *BitSet) Union(c, d *BitSet) *BitSet {
	if b == nil {
		b = new(BitSet)
	}
	if len(c.bits) < len(d.bits) {
		c, d = d, c
	}
	cbits, dbits := c.bits, d.bits
	if len(c.bits) > len(b.bits) {
		b.bits = make([]uint64, len(c.bits))
	}
	for i := range b.bits {
		b.bits[i] = cbits[i] | dbits[i]
	}
	return b
}

func (b *BitSet) Intersect(c, d *BitSet) *BitSet {
	if b == nil {
		b = new(BitSet)
	}
	if len(c.bits) > len(d.bits) {
		c, d = d, c
	}
	cbits, dbits := c.bits, d.bits
	if len(c.bits) > len(b.bits) {
		b.bits = make([]uint64, len(c.bits))
	}
	for i := range b.bits {
		b.bits[i] = cbits[i] & dbits[i]
	}
	return b
}

func (b *BitSet) Add(x Int) {
	index := int(x >> wbits)
	if index >= len(b.bits) {
		bits := make([]uint64, index+1)
		copy(bits, b.bits)
		b.bits = bits
	}
	b.bits[index] |= 1 << (x & (wbits - 1))
}

func (b *BitSet) Remove(x Int) {
	panic("unimplemented")
}

func (b *BitSet) Iter() Iter[Int] {
	return &bitIter{
		bits:  b.bits,
		index: -1,
	}
}

type bitIter struct {
	bits  []uint64
	index int
}

func (iter *bitIter) Next() bool {
	panic("unimplemented")
}

func (iter *bitIter) Item() Int {
	return Int(iter.index)
}

// Verify that BitSet implements Set.
var _ Set[*BitSet, Int] = (*BitSet)(nil)
