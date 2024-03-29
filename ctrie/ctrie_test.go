/*
Copyright 2015 Workiva, LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ctrie

import (
	"bytes"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCtrie(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, string](bytes.Equal, BytesHash)

	_, ok := ctrie.Get([]byte("foo"))
	assertFalse(t, ok)

	ctrie.Set([]byte("foo"), "bar")
	val, ok := ctrie.Get([]byte("foo"))
	assertTrue(t, ok)
	assertEqual(t, "bar", val)

	ctrie.Set([]byte("fooooo"), "baz")
	val, ok = ctrie.Get([]byte("foo"))
	assertTrue(t, ok)
	assertEqual(t, "bar", val)
	val, ok = ctrie.Get([]byte("fooooo"))
	assertTrue(t, ok)
	assertEqual(t, "baz", val)

	for i := 0; i < 100; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), "blah")
	}
	for i := 0; i < 100; i++ {
		val, ok = ctrie.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, "blah", val)
	}

	val, ok = ctrie.Get([]byte("foo"))
	assertTrue(t, ok)
	assertEqual(t, "bar", val)
	ctrie.Set([]byte("foo"), "qux")
	val, ok = ctrie.Get([]byte("foo"))
	assertTrue(t, ok)
	assertEqual(t, "qux", val)

	val, ok = ctrie.Delete([]byte("foo"))
	assertTrue(t, ok)
	assertEqual(t, "qux", val)

	_, ok = ctrie.Delete([]byte("foo"))
	assertFalse(t, ok)

	val, ok = ctrie.Delete([]byte("fooooo"))
	assertTrue(t, ok)
	assertEqual(t, "baz", val)

	for i := 0; i < 100; i++ {
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}
}

func TestSetLNode(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, func([]byte) uint64 { return 0 })

	for i := 0; i < 10; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}

	for i := 0; i < 10; i++ {
		val, ok := ctrie.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}
	_, ok := ctrie.Get([]byte("11"))
	assertFalse(t, ok)

	for i := 0; i < 10; i++ {
		val, ok := ctrie.Delete([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}
}

func TestSetTNode(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)

	for i := 0; i < 10000; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}

	for i := 0; i < 5000; i++ {
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}

	for i := 0; i < 10000; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}

	for i := 0; i < 10000; i++ {
		val, ok := ctrie.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}
}

func TestConcurrency(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < 10000; i++ {
			ctrie.Set([]byte(strconv.Itoa(i)), i)
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 10000; i++ {
			val, ok := ctrie.Get([]byte(strconv.Itoa(i)))
			if ok {
				assertEqual(t, i, val)
			}
		}
		wg.Done()
	}()

	for i := 0; i < 10000; i++ {
		time.Sleep(5)
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}

	wg.Wait()
}

func TestConcurrency2(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		for i := 0; i < 10000; i++ {
			ctrie.Set([]byte(strconv.Itoa(i)), i)
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 10000; i++ {
			val, ok := ctrie.Get([]byte(strconv.Itoa(i)))
			if ok {
				assertEqual(t, i, val)
			}
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 10000; i++ {
			ctrie.Clone()
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 10000; i++ {
			ctrie.RClone()
		}
		wg.Done()
	}()

	wg.Wait()
	assertEqual(t, 10000, ctrie.Len())
}

func TestClone(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < 100; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}

	snapshot := ctrie.Clone()

	// Ensure snapshot contains expected keys.
	for i := 0; i < 100; i++ {
		val, ok := snapshot.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}

	// Now remove the values from the original.
	for i := 0; i < 100; i++ {
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}

	// Ensure snapshot was unaffected by removals.
	for i := 0; i < 100; i++ {
		val, ok := snapshot.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}

	// New Ctrie and snapshot.
	ctrie = NewWithFuncs[[]byte, int](nil, nil)
	for i := 0; i < 100; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	snapshot = ctrie.Clone()

	// Ensure snapshot is mutable.
	for i := 0; i < 100; i++ {
		snapshot.Delete([]byte(strconv.Itoa(i)))
	}
	snapshot.Set([]byte("bat"), 5000)

	for i := 0; i < 100; i++ {
		_, ok := snapshot.Get([]byte(strconv.Itoa(i)))
		assertFalse(t, ok)
	}
	val, ok := snapshot.Get([]byte("bat"))
	assertTrue(t, ok)
	assertEqual(t, 5000, val)

	// Ensure original Ctrie was unaffected.
	for i := 0; i < 100; i++ {
		val, ok := ctrie.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}
	_, ok = ctrie.Get([]byte("bat"))
	assertFalse(t, ok)

	// Ensure snapshots-of-snapshots work as expected.
	snapshot2 := snapshot.Clone()
	for i := 0; i < 100; i++ {
		_, ok := snapshot2.Get([]byte(strconv.Itoa(i)))
		assertFalse(t, ok)
	}
	val, ok = snapshot2.Get([]byte("bat"))
	assertTrue(t, ok)
	assertEqual(t, 5000, val)

	snapshot2.Delete([]byte("bat"))
	_, ok = snapshot2.Get([]byte("bat"))
	assertFalse(t, ok)
	val, ok = snapshot.Get([]byte("bat"))
	assertTrue(t, ok)
	assertEqual(t, 5000, val)
}

func TestRClone(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](nil, nil)
	for i := 0; i < 100; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}

	snapshot := ctrie.RClone()

	// Ensure snapshot contains expected keys.
	for i := 0; i < 100; i++ {
		val, ok := snapshot.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}

	for i := 0; i < 50; i++ {
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}

	// Ensure snapshot was unaffected by removals.
	for i := 0; i < 100; i++ {
		val, ok := snapshot.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}

	// Ensure read-only snapshots panic on writes.
	func() {
		defer func() {
			assertNotNil(t, recover())
		}()
		snapshot.Delete([]byte("blah"))
	}()

	// Ensure snapshots-of-snapshots work as expected.
	snapshot2 := snapshot.Clone()
	for i := 50; i < 100; i++ {
		ctrie.Delete([]byte(strconv.Itoa(i)))
	}
	for i := 0; i < 100; i++ {
		val, ok := snapshot2.Get([]byte(strconv.Itoa(i)))
		assertTrue(t, ok)
		assertEqual(t, i, val)
	}

	// Ensure snapshots of read-only snapshots panic on writes.
	func() {
		defer func() {
			assertNotNil(t, recover())
		}()
		snapshot2.Delete([]byte("blah"))
	}()
}

func TestIterator(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](nil, nil)
	for i := 0; i < 10; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	expected := map[string]int{
		"0": 0,
		"1": 1,
		"2": 2,
		"3": 3,
		"4": 4,
		"5": 5,
		"6": 6,
		"7": 7,
		"8": 8,
		"9": 9,
	}

	count := 0
	for iter := ctrie.Iterator(); iter.Next(); {
		exp, ok := expected[string(iter.Key())]
		if assertTrue(t, ok) {
			assertEqual(t, exp, iter.Value())
		}
		count++
	}
	assertEqual(t, len(expected), count)
}

// TestIteratorCoversTNodes reproduces the scenario of a bug where tNodes weren't being traversed.
func TestIteratorCoversTNodes(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, bool](nil, func([]byte) uint64 { return 0 })
	// Add a pair of keys that collide (because we're using the mock hash).
	ctrie.Set([]byte("a"), true)
	ctrie.Set([]byte("b"), true)
	// Delete one key, leaving exactly one sNode in the cNode.  This will
	// trigger creation of a tNode.
	ctrie.Delete([]byte("b"))
	seenKeys := map[string]bool{}
	for iter := ctrie.Iterator(); iter.Next(); {
		seenKeys[string(iter.Key())] = true
	}
	if !seenKeys["a"] {
		t.Errorf("Iterator did not return 'a'.")
	}
	if len(seenKeys) != 1 {
		t.Errorf("want 1 key got %d", len(seenKeys))
	}
}

func TestLen(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < 10; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	assertEqual(t, 10, ctrie.Len())
}

func TestClear(t *testing.T) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < 10; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	assertEqual(t, 10, ctrie.Len())
	snapshot := ctrie.Clone()

	ctrie.Clear()

	assertEqual(t, 0, ctrie.Len())
	assertEqual(t, 10, snapshot.Len())
}

func TestHashCollision(t *testing.T) {
	trie := NewWithFuncs[[]byte, int](bytes.Equal, func([]byte) uint64 {
		return 42
	})
	trie.Set([]byte("foobar"), 1)
	trie.Set([]byte("zogzog"), 2)
	trie.Set([]byte("foobar"), 3)
	val, exists := trie.Get([]byte("foobar"))
	assertTrue(t, exists)
	assertEqual(t, 3, val)

	trie.Delete([]byte("foobar"))

	_, exists = trie.Get([]byte("foobar"))
	assertFalse(t, exists)
}

func BenchmarkSet(b *testing.B) {
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrie.Set([]byte("foo"), 0)
	}
}

func BenchmarkGet(b *testing.B) {
	numItems := 1000
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < numItems; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	key := []byte(strconv.Itoa(numItems / 2))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrie.Get(key)
	}
}

func BenchmarkDelete(b *testing.B) {
	numItems := 1000
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < numItems; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	key := []byte(strconv.Itoa(numItems / 2))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrie.Delete(key)
	}
}

func BenchmarkClone(b *testing.B) {
	numItems := 1000
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < numItems; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrie.Clone()
	}
}

func BenchmarkRClone(b *testing.B) {
	numItems := 1000
	ctrie := NewWithFuncs[[]byte, int](bytes.Equal, BytesHash)
	for i := 0; i < numItems; i++ {
		ctrie.Set([]byte(strconv.Itoa(i)), i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctrie.RClone()
	}
}

func assertTrue(t *testing.T, x bool) bool {
	t.Helper()
	if !x {
		t.Errorf("not true")
		return false
	}
	return true
}

func assertFalse(t *testing.T, x bool) {
	t.Helper()
	if x {
		t.Errorf("not false")
	}
}

func assertEqual[T comparable](t *testing.T, x, y T) {
	t.Helper()
	if x != y {
		t.Errorf("not equal, got %#v want %#v", y, x)
	}
}

func assertNotNil(t *testing.T, x interface{}) {
	t.Helper()
	if x == nil {
		t.Errorf("want non-nil, got nil")
	}
}
