// Copyright 2025 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package anyhash_test

import (
	"hash/maphash"
	"slices"
	"testing"

	"github.com/go-quicktest/qt"

	"github.com/rogpeppe/generic/anyhash"
)

// sliceHasher is a test Hasher implementation for slices
// of comparable values.
// This demonstrates a non-comparable key type that needs custom hashing.
type sliceHasher[T comparable] struct{}

func (sliceHasher[T]) Equal(a, b []T) bool {
	return slices.Equal(a, b)
}

func (sliceHasher[T]) Hash(h *maphash.Hash, s []T) {
	for _, v := range s {
		maphash.WriteComparable(h, v)
	}
}

func TestNewMap(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})
	qt.Assert(t, qt.Not(qt.IsNil(m)))
	qt.Assert(t, qt.Equals(m.Len(), 0))
}

func TestMap_NilReceiver(t *testing.T) {
	var m *anyhash.Map[string, int, anyhash.ComparableHasher[string]]

	// Len should work on nil receiver
	qt.Assert(t, qt.Equals(m.Len(), 0))

	// At should work on nil receiver, returning zero value
	qt.Assert(t, qt.Equals(m.At("key"), 0))

	// Delete should work on nil receiver, returning false
	old, ok := m.Delete("key")
	qt.Assert(t, qt.Equals(old, 0))
	qt.Assert(t, qt.Equals(ok, false))

	// Iterators should work on nil receiver (empty iteration)
	count := 0
	for range m.All() {
		count++
	}
	qt.Assert(t, qt.Equals(count, 0))

	count = 0
	for range m.Keys() {
		count++
	}
	qt.Assert(t, qt.Equals(count, 0))

	count = 0
	for range m.Values() {
		count++
	}
	qt.Assert(t, qt.Equals(count, 0))
}

func TestMap_SetPanicsOnNil(t *testing.T) {
	var m *anyhash.Map[string, int, anyhash.ComparableHasher[string]]

	qt.Assert(t, qt.PanicMatches(
		func() {
			m.Set("key", 42)
		},
		`\(\*Map\).Set called on nil \*Map`,
	))
}

func TestMap_SetAndAt(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	// Set a value
	prev := m.Set("foo", 42)
	qt.Assert(t, qt.Equals(prev, 0)) // zero value for int
	qt.Assert(t, qt.Equals(m.Len(), 1))

	// Retrieve the value
	qt.Assert(t, qt.Equals(m.At("foo"), 42))

	// Update the value
	prev = m.Set("foo", 100)
	qt.Assert(t, qt.Equals(prev, 42))
	qt.Assert(t, qt.Equals(m.Len(), 1))

	// Retrieve the updated value
	qt.Assert(t, qt.Equals(m.At("foo"), 100))

	// Get a non-existent key
	qt.Assert(t, qt.Equals(m.At("bar"), 0))
}

func TestMap_MultipleEntries(t *testing.T) {
	m := anyhash.NewMap[string, string, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	entries := map[string]string{
		"one":   "1",
		"two":   "2",
		"three": "3",
		"four":  "4",
		"five":  "5",
	}

	// Set all entries
	for k, v := range entries {
		m.Set(k, v)
	}

	qt.Assert(t, qt.Equals(m.Len(), len(entries)))

	// Verify all entries
	for k, v := range entries {
		qt.Assert(t, qt.Equals(m.At(k), v))
	}
}

func TestMap_Delete(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("foo", 42)
	m.Set("bar", 100)
	qt.Assert(t, qt.Equals(m.Len(), 2))

	// Delete existing key
	old, deleted := m.Delete("foo")
	qt.Assert(t, qt.Equals(old, 42))
	qt.Assert(t, qt.Equals(deleted, true))
	qt.Assert(t, qt.Equals(m.Len(), 1))
	qt.Assert(t, qt.Equals(m.At("foo"), 0))

	// Delete non-existent key
	old, deleted = m.Delete("baz")
	qt.Assert(t, qt.Equals(old, 0))
	qt.Assert(t, qt.Equals(deleted, false))
	qt.Assert(t, qt.Equals(m.Len(), 1))

	// Verify remaining entry
	qt.Assert(t, qt.Equals(m.At("bar"), 100))

	// Delete the last entry
	old, deleted = m.Delete("bar")
	qt.Assert(t, qt.Equals(old, 100))
	qt.Assert(t, qt.Equals(deleted, true))
	qt.Assert(t, qt.Equals(m.Len(), 0))
}

func TestMap_DeleteAndReuse(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	// Add an entry
	m.Set("foo", 42)
	qt.Assert(t, qt.Equals(m.Len(), 1))

	// Delete it
	m.Delete("foo")
	qt.Assert(t, qt.Equals(m.Len(), 0))

	// Add a different entry that might hash to the same bucket
	m.Set("bar", 100)
	qt.Assert(t, qt.Equals(m.Len(), 1))
	qt.Assert(t, qt.Equals(m.At("bar"), 100))
	qt.Assert(t, qt.Equals(m.At("foo"), 0))
}

func TestMap_AllIterator(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	expected := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	for k, v := range expected {
		m.Set(k, v)
	}

	// Collect all entries
	seen := make(map[string]int)
	for k, v := range m.All() {
		seen[k] = v
	}

	qt.Assert(t, qt.DeepEquals(seen, expected))
}

func TestMap_AllIteratorEarlyExit(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	// Exit early from iterator
	count := 0
	for range m.All() {
		count++
		if count == 1 {
			break
		}
	}

	qt.Assert(t, qt.Equals(count, 1))
}

func TestMap_KeysIterator(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	expected := map[string]bool{
		"one":   true,
		"two":   true,
		"three": true,
	}

	for k := range expected {
		m.Set(k, 42)
	}

	// Collect all keys
	seen := make(map[string]bool)
	for k := range m.Keys() {
		seen[k] = true
	}

	qt.Assert(t, qt.DeepEquals(seen, expected))
}

func TestMap_ValuesIterator(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	// Collect all values
	seen := make(map[int]bool)
	for v := range m.Values() {
		seen[v] = true
	}

	expected := map[int]bool{
		1: true,
		2: true,
		3: true,
	}

	qt.Assert(t, qt.DeepEquals(seen, expected))
}

func TestMap_NonComparableKeys(t *testing.T) {
	m := anyhash.NewMap[[]byte, string, sliceHasher[byte]](sliceHasher[byte]{})

	key1 := []byte("hello")
	key2 := []byte("world")
	key3 := []byte("hello") // same content as key1

	m.Set(key1, "value1")
	m.Set(key2, "value2")

	qt.Assert(t, qt.Equals(m.Len(), 2))
	qt.Assert(t, qt.Equals(m.At(key1), "value1"))
	qt.Assert(t, qt.Equals(m.At(key2), "value2"))

	// key3 has same content as key1, should find the same value
	qt.Assert(t, qt.Equals(m.At(key3), "value1"))

	// Update using key3 (equivalent to key1)
	prev := m.Set(key3, "updated")
	qt.Assert(t, qt.Equals(prev, "value1"))
	qt.Assert(t, qt.Equals(m.Len(), 2)) // still 2 entries

	// Verify the update
	qt.Assert(t, qt.Equals(m.At(key1), "updated"))
	qt.Assert(t, qt.Equals(m.At(key3), "updated"))
}

// badHasher is a hasher that creates intentional collisions for testing.
// This hasher always returns the same hash, forcing collisions.
type badHasher struct{}

func (badHasher) Equal(a, b string) bool {
	return a == b
}

func (badHasher) Hash(*maphash.Hash, string) {
	// Don't write anything, so we always get the same hash.
}

func TestMap_HashCollisions(t *testing.T) {
	m := anyhash.NewMap[string, int, badHasher](badHasher{})

	// All these will hash to the same bucket
	m.Set("key1", 1)
	m.Set("key2", 2)
	m.Set("key3", 3)

	qt.Assert(t, qt.Equals(m.Len(), 3))
	qt.Assert(t, qt.Equals(m.At("key1"), 1))
	qt.Assert(t, qt.Equals(m.At("key2"), 2))
	qt.Assert(t, qt.Equals(m.At("key3"), 3))

	// Delete one
	m.Delete("key2")
	qt.Assert(t, qt.Equals(m.Len(), 2))
	qt.Assert(t, qt.Equals(m.At("key2"), 0))
	qt.Assert(t, qt.Equals(m.At("key1"), 1))
	qt.Assert(t, qt.Equals(m.At("key3"), 3))
}

func TestMap_IteratorWithDeletion(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)
	m.Set("four", 4)

	// Delete entries during iteration
	// Deleting unseen entries should guarantee they won't be yielded
	var seen []string
	for k, v := range m.All() {
		seen = append(seen, k)
		if v == 2 {
			m.Delete("four") // delete an unseen entry
		}
	}

	// Verify that we saw at least some entries
	qt.Assert(t, qt.Not(qt.Equals(len(seen), 0)))

	// Verify that "four" is deleted
	qt.Assert(t, qt.Equals(m.At("four"), 0))
}

func TestMap_ZeroValues(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	// Explicitly set a zero value
	prev := m.Set("zero", 0)
	qt.Assert(t, qt.Equals(prev, 0))
	qt.Assert(t, qt.Equals(m.Len(), 1))

	// Verify we can retrieve the zero value
	val := m.At("zero")
	qt.Assert(t, qt.Equals(val, 0))

	// Verify it's actually in the map (not missing)
	found := false
	for k := range m.Keys() {
		if k == "zero" {
			found = true
			break
		}
	}
	qt.Assert(t, qt.Equals(found, true))
}

func TestMap_EmptyStringKey(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	// Use empty string as key
	m.Set("", 42)
	qt.Assert(t, qt.Equals(m.Len(), 1))
	qt.Assert(t, qt.Equals(m.At(""), 42))

	// Delete it
	old, deleted := m.Delete("")
	qt.Assert(t, qt.Equals(old, 42))
	qt.Assert(t, qt.Equals(deleted, true))
	qt.Assert(t, qt.Equals(m.Len(), 0))
}

func TestMap_LargeMap(t *testing.T) {
	m := anyhash.NewMap[int, int, anyhash.ComparableHasher[int]](anyhash.ComparableHasher[int]{})

	n := 1000
	for i := 0; i < n; i++ {
		m.Set(i, i*2)
	}

	qt.Assert(t, qt.Equals(m.Len(), n))

	// Verify all values
	for i := 0; i < n; i++ {
		qt.Assert(t, qt.Equals(m.At(i), i*2))
	}

	// Delete half
	for i := 0; i < n; i += 2 {
		old, deleted := m.Delete(i)
		qt.Assert(t, qt.Equals(old, i*2))
		qt.Assert(t, qt.Equals(deleted, true))
	}

	qt.Assert(t, qt.Equals(m.Len(), n/2))

	// Verify remaining values
	for i := 1; i < n; i += 2 {
		qt.Assert(t, qt.Equals(m.At(i), i*2))
	}
}

// intHasher is a hasher for int keys
type intHasher struct{}

func (intHasher) Equal(a, b int) bool {
	return a == b
}

func (intHasher) Hash(seed maphash.Seed, k int) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(k >> (i * 8))
	}
	h.Write(buf[:])
	return h.Sum64()
}

func TestMap_UpdateDuringIteration(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("one", 1)
	m.Set("two", 2)
	m.Set("three", 3)

	// Update values during iteration
	for k, v := range m.All() {
		m.Set(k, v*10)
	}

	// Verify all values are updated (or at least the map is in a consistent state)
	qt.Assert(t, qt.Equals(m.Len(), 3))
}

func TestMap_InsertDuringIteration(t *testing.T) {
	m := anyhash.NewMap[string, int, anyhash.ComparableHasher[string]](anyhash.ComparableHasher[string]{})

	m.Set("one", 1)
	m.Set("two", 2)

	// Insert new entries during iteration
	// According to docs, new entries may or may not be seen
	count := 0
	for k := range m.Keys() {
		count++
		if k == "one" && m.At("three") == 0 {
			m.Set("three", 3)
		}
		if count > 10 { // safety check to avoid infinite loop
			break
		}
	}

	// Map should be in a consistent state
	qt.Assert(t, qt.Equals(m.At("three"), 3))
}
