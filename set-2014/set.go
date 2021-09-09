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
	self Set[self, elem],
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
	bits []uintptr
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
	panic("unimplemented")
}

func (b *BitSet) Intersect(c, d *BitSet) *BitSet {
	panic("unimplemented")
}

func (b *BitSet) Add(x Int) {
	b.bits[x>>wbits] |= 1 << (x & (wbits - 1))
}

func (b *BitSet) Remove(x Int) {
	panic("unimplemented")
}

func (b *BitSet) Iter() Iter[Int] {
	panic("unimplemented")
}

// Verify that BitSet implements Set.
var _ Set[*BitSet, Int] = (*BitSet)(nil)
