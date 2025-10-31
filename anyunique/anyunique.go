// Package anyunique provides canonicalization of values under a
// caller-defined equivalence relation.
//
// A [Set] holds a set of unique values of a specific type T. Calling
// [Set.Make] with two values that are equivalent according to the provided
// [Hasher] returns [Handle] values that are identical. [Handle] is a lightweight
// wrapper around the canonical value; use [Handle.Value] to obtain the
// underlying T.
//
// The zero [Handle] represents the zero value of T. Make returns the zero
// [Handle] when called with the zero value of T: it will never try to hash
// the zero value.
//
// [Set.WriteHash] writes a short representation of a canonicalized
// value to a [maphash.Hash]. It is useful when hashing structures that
// themselves contain canonicalized values, avoiding re-hashing the full
// value graph.
//
// Values in a set are amenable to garbage collection: the set
// does not necesarily always grow in size.
//
// NOTE this package assumes that T values are treated as immutable.
// That is, after calling [Set.Make] a value must not change.
package anyunique

import (
	"hash/maphash"
	"sync"
	"weak"
)

// A Hasher defines a hash function and an equivalence relation over
// values of type T.
//
// Hash must write a hash of its argument to the provided *maphash.Hash,
// and Equal must report whether two values are equivalent. Hash and
// Equal must be consistent: if Equal(x, y) is true then Hash must
// produce the same output for x and y.
//
// See https://go-review.googlesource.com/c/go/+/657296/11/src/hash/maphash/hasher.go
type Hasher[T any] interface {
	comparable
	Hash(*maphash.Hash, T)
	Equal(x, y T) bool
}

type emptyPair[a, b any] struct{}

// (T, H) -> *Set[T, H]
var cache sync.Map

// New returns a new set holding unique values
// of type T, using h to determine whether values are the
// same.
//
// The equivalence relation and hash are supplied by the given [Hasher].
func New[T any, H Hasher[T]](h H) *Set[T, H] {
	cacheKey := emptyPair[T, H]{}
	entry, ok := cache.Load(cacheKey)
	if ok {
		return entry.(*Set[T, H])
	}

	s := &Set[T, H]{
		h:       h,
		seed:    maphash.MakeSeed(),
		entries: make(map[uint64][]weak.Pointer[value[T]]),
	}
	// Find out the hash for the zero value.
	s.zeroHash = s.hashOf(*new(T))
	if h == *new(H) {
		// The hash is zero-valued (common case: no state), so stash it.
		s1, _ := cache.LoadOrStore(cacheKey, s)
		s = s1.(*Set[T, H])
	}
	return s
}

// Set holds a set of unique values of type T.
type Set[T any, H Hasher[T]] struct {
	h        H
	seed     maphash.Seed
	zeroHash uint64
	entries  map[uint64][]weak.Pointer[value[T]]
}

// Handle represents a unique value of type T. If two values of type
// Handle[T] originating from the same [Set] compare equal, they are
// guaranteed to be equal according to the equality criteria that the
// set was created with.
type Handle[T any] struct {
	x *value[T]
}

type value[T any] struct {
	x    T
	hash uint64
}

// Get returns the actual value held in u. The zero value
// of [Handle] returns the zero value of T.
func (u Handle[T]) Value() T {
	if u.x == nil {
		// We know it's OK to return the zero T for the
		// zero U because we always add an entry for the
		// zero T when the set is created.
		return *new(T)
	}
	return u.x.x
}

// WriteHash writes a short representation of u to h.
// This allows callers to avoid hashing an tree of values
// when hashing a value that itself contains other Handle[T] items.
func (u Handle[T]) WriteHash(h *maphash.Hash) {
	if u.x == nil {
		// We don't know what the actual hash of the
		// zero value is, but it doesn't matter - we just
		// use an arbitrary consistent value so we don't need to consult
		// the actual set.
		maphash.WriteComparable(h, 0)
		return
	}
	// TODO we _could_ write two independent hashes here
	// if we were concerned about collisions.
	maphash.WriteComparable(h, u.x.hash)
}

// Make returns a unique value u such that u.Get() is
// equal to x according to the equality criteria
// defined by the set.
//
// It is assumed that values will not change after
// passing to Make: the caller must take care to
// preserve immutability.
func (s *Set[T, H]) Make(x T) Handle[T] {
	h := s.hashOf(x)
	if h == s.zeroHash && s.h.Equal(x, *new(T)) {
		// Anything comparing equal to the zero T
		// turns into the zero U.
		return Handle[T]{}
	}
	entries := s.entries[h]
	firstEmpty := -1
	for i, ep := range entries {
		if e := ep.Value(); e != nil {
			if s.h.Equal(x, e.x) {
				return Handle[T]{e}
			}
		} else if firstEmpty == -1 {
			firstEmpty = i
		}
	}
	v := value[T]{x, h}
	// TODO we could use runtime.AddCleanup to remove
	// the entry from the map if it's the last one removed.
	entry := weak.Make(&v)
	if firstEmpty != -1 {
		entries[firstEmpty] = entry
	} else {
		s.entries[h] = append(entries, entry)
	}
	return Handle[T]{&v}
}

func (s *Set[T, H]) hashOf(x T) uint64 {
	var hasher maphash.Hash
	hasher.SetSeed(s.seed)
	s.h.Hash(&hasher, x)
	return hasher.Sum64()
}
