
// Package anyhash implements support for storing a map
// storing arbitrary hashable values that aren't necessarily
// comparable.
package anyhash

import (
	"hash/maphash"
	"iter"
)

// See https://go-review.googlesource.com/c/go/+/657296/11/src/hash/maphash/hasher.go#7

// A Hasher defines a hash function and an equivalence relation over
// values of type T.
//
// See https://go-review.googlesource.com/c/go/+/657296/11/src/hash/maphash/hasher.go
type Hasher[T any] interface {
	Hash(*maphash.Hash, T)
	Equal(x, y T) bool
}

// ComparableHasher is an implementation of [Hasher] for comparable types.
// Its Equal(x, y) method is consistent with x == y.
type ComparableHasher[T comparable] struct {
	_ [0]func(T) // disallow comparison, and conversion between ComparableHasher[X] and ComparableHasher[Y]
}

func (ComparableHasher[T]) Hash(h *maphash.Hash, v T) { maphash.WriteComparable(h, v) }
func (ComparableHasher[T]) Equal(x, y T) bool         { return x == y }

// Map is a hash-table-based mapping from keys K to values V,
// parameterized by a stateless hasher/equality provider H.
//
// Just as with map[K]V, a nil *Map is a valid empty map.
//
// Read-only operations (At, Len, All/Keys/Values, String) may be called
// concurrently with each other, but this type does not provide external
// synchronization for concurrent mutation.
type Map[K, V any, H Hasher[K]] struct {
	hasher Hasher[K]
	seed   maphash.Seed
	table  map[uint64][]entry[K, V] // maps hash to bucket; entry.key==zero means unused (tracked via used flag)
	length int
}

// entry is an association in a hash bucket.
type entry[K, V any] struct {
	key  K
	val  V
	used bool // distinguishes empty slot from zero K/V
}

// NewMap returns a new empty Map.
func NewMap[K, V any, H Hasher[K]](h Hasher[K]) *Map[K, V, H] {
	return &Map[K, V, H]{
		hasher: h,
		seed:   maphash.MakeSeed(),
		table:  make(map[uint64][]entry[K, V]),
	}
}

// Len returns the number of entries in the map.
func (m *Map[K, V, H]) Len() int {
	if m == nil {
		return 0
	}
	return m.length
}

func (m *Map[K, V, H]) hashKey(k K) uint64 {
	var h maphash.Hash
	h.SetSeed(m.seed)
	m.hasher.Hash(&h, k)
	return h.Sum64()
}

// find locates the bucket and index for key k, if present.
// Returns (bucket, index, found).
func (m *Map[K, V, H]) find(k K) ([]entry[K, V], int, bool) {
	if m == nil || m.table == nil {
		return nil, -1, false
	}
	b := m.table[m.hashKey(k)]
	for i := range b {
		if b[i].used && m.hasher.Equal(k, b[i].key) {
			return b, i, true
		}
	}
	return b, -1, false
}

// At returns the value for key k, or the zero value of V if not present.
func (m *Map[K, V, H]) At(k K) (v V) {
	if b, i, ok := m.find(k); ok {
		return b[i].val
	}
	return *new(V)
}

// Get returns the key stored in the map (Equal to k but not necessarily
// exactly the same), its associated value, and reports
// whether the entry was found.
func (m *Map[K, V, H]) Get(k K) (K, V, bool) {
	if b, i, ok := m.find(k); ok {
		e := b[i]
		return e.key, e.val, true
	}
	return *new(K), *new(V), false
}

// Set sets the value for k to v, returning the previous value (or zero if none).
func (m *Map[K, V, H]) Set(k K, v V) (prev V) {
	if m == nil {
		// Allow calling on nil receiver by converting to zero map then growing.
		panic("(*Map).Set called on nil *Map")
	}

	if m.table == nil {
		m.table = make(map[uint64][]entry[K, V])
	}

	hv := m.hashKey(k)
	b := m.table[hv]

	// Track first hole for potential reuse
	hole := -1

	for i := range b {
		used := b[i].used
		if !used && hole == -1 {
			hole = i
			continue
		}
		if used && m.hasher.Equal(k, b[i].key) {
			prev = b[i].val
			b[i].val = v
			return prev
		}
	}

	if hole != -1 {
		b[hole] = entry[K, V]{key: k, val: v, used: true}
	} else {
		m.table[hv] = append(b, entry[K, V]{key: k, val: v, used: true})
	}
	m.length++
	return prev
}

// Delete removes the entry with key k, if present, and reports whether it was found.
func (m *Map[K, V, H]) Delete(k K) (old V, deleted bool) {
	if m == nil || m.table == nil {
		return *new(V), false
	}
	hv := m.hashKey(k)
	b := m.table[hv]
	for i := range b {
		if b[i].used && m.hasher.Equal(k, b[i].key) {
			// Do not compact to preserve iterator behavior.
			old = b[i].val
			b[i] = entry[K, V]{}
			m.length--
			return old, true
		}
	}
	return *new(V), false
}

// All returns an iterator over (key, value) pairs in unspecified order.
//
// If the caller mutates the map while iterating, the usual Go map-style
// caveats apply: deleting an unseen entry guarantees it won't be yielded;
// inserting a new entry may or may not be seen by the iterator.
func (m *Map[K, V, H]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if m == nil || m.table == nil {
			return
		}
		for _, bucket := range m.table {
			for i := range bucket {
				if bucket[i].used {
					if !yield(bucket[i].key, bucket[i].val) {
						return
					}
				}
			}
		}
	}
}

// Keys returns an iterator over keys in unspecified order.
func (m *Map[K, V, H]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		if m == nil || m.table == nil {
			return
		}
		for _, bucket := range m.table {
			for i := range bucket {
				if bucket[i].used {
					if !yield(bucket[i].key) {
						return
					}
				}
			}
		}
	}
}

// Values returns an iterator over values in unspecified order.
func (m *Map[K, V, H]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		if m == nil || m.table == nil {
			return
		}
		for _, bucket := range m.table {
			for i := range bucket {
				if bucket[i].used {
					if !yield(bucket[i].val) {
						return
					}
				}
			}
		}
	}
}