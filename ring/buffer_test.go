package ring_test

import (
	"reflect"
	"testing"

	"github.com/rogpeppe/generic/ring"
)

func TestEmptyBuffer(t *testing.T) {
	b := ring.NewBuffer[int](10) // Assume a constructor that creates a buffer with capacity 10

	if got := b.Len(); got != 0 {
		t.Errorf("expected Len = 0, got %d", got)
	}

	// Operations that should panic or fail on empty buffer
	mustPanic(t, func() { b.PopStart() })
	mustPanic(t, func() { b.PopEnd() })
	mustPanic(t, func() { b.PeekStart() })
	mustPanic(t, func() { b.PeekEnd() })
	mustPanic(t, func() { b.Get(0) })
}

func TestSingleElementPushPop(t *testing.T) {
	b := ring.NewBuffer[string](5)

	b.PushStart("A")
	if b.Len() != 1 {
		t.Errorf("expected length = 1 after PushStart, got %d", b.Len())
	}
	t.Logf("%v", b)
	if b.PeekStart() != "A" || b.PeekEnd() != "A" {
		t.Errorf("PeekStart or PeekEnd did not return the single pushed value.")
	}

	b.PushEnd("B")
	if b.Len() != 2 {
		t.Errorf("expected length = 2 after PushEnd, got %d", b.Len())
	}
	if got := b.PeekEnd(); got != "B" {
		t.Errorf("expected PeekEnd = B, got %v", got)
	}

	// Test Get
	if got := b.Get(0); got != "A" {
		t.Errorf("Get(0) = %v; want A", got)
	}
	if got := b.Get(1); got != "B" {
		t.Errorf("Get(1) = %v; want B", got)
	}

	popped := b.PopStart()
	if popped != "A" {
		t.Errorf("PopStart() = %v; want A", popped)
	}
	if b.Len() != 1 {
		t.Errorf("expected Len = 1 after PopStart, got %d", b.Len())
	}

	popped = b.PopEnd()
	if popped != "B" {
		t.Errorf("PopEnd() = %v; want B", popped)
	}
	if b.Len() != 0 {
		t.Errorf("expected Len = 0 after PopEnd, got %d", b.Len())
	}
}

func TestPushMultiple(t *testing.T) {
	b := ring.NewBuffer[int](5)

	// Fill up buffer
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)
	b.PushEnd(4)
	b.PushEnd(5)

	if b.Len() != 5 {
		t.Errorf("expected full length = 5, got %d", b.Len())
	}

	// Check indexing
	for i := 0; i < 5; i++ {
		if got := b.Get(i); got != i+1 {
			t.Errorf("Get(%d) = %d; want %d", i, got, i+1)
		}
	}

	// Discard some from start and end
	discarded := b.DiscardFromStart(2)
	if discarded != 2 {
		t.Errorf("DiscardFromStart(2) = %d; want 2", discarded)
	}
	if b.Len() != 3 {
		t.Errorf("Len after discarding 2 from start = %d; want 3", b.Len())
	}
	// Remaining should be [3,4,5]

	discarded = b.DiscardFromEnd(1)
	if discarded != 1 {
		t.Errorf("DiscardFromEnd(1) = %d; want 1", discarded)
	}
	if b.Len() != 2 {
		t.Errorf("Len after discarding 1 from end = %d; want 2", b.Len())
	}
	// Remaining should be [3,4]

	if got := b.PeekStart(); got != 3 {
		t.Errorf("PeekStart = %d; want 3", got)
	}
	if got := b.PeekEnd(); got != 4 {
		t.Errorf("PeekEnd = %d; want 4", got)
	}
}

func TestPopUntilEmpty(t *testing.T) {
	b := ring.NewBuffer[int](3)
	b.PushEnd(10)
	b.PushEnd(20)
	b.PushEnd(30)

	if b.PopStart() != 10 {
		t.Error("PopStart expected 10")
	}
	if b.PopStart() != 20 {
		t.Error("PopStart expected 20")
	}
	if b.PopStart() != 30 {
		t.Error("PopStart expected 30")
	}

	if b.Len() != 0 {
		t.Errorf("Buffer should be empty, length = %d", b.Len())
	}

	// Verify that popping now panics
	mustPanic(t, func() { b.PopStart() })
	mustPanic(t, func() { b.PopEnd() })
}

func TestWrapAround(t *testing.T) {
	b := ring.NewBuffer[int](3)
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)

	// Now buffer is full: [1,2,3]
	// Pop one from the start
	got := b.PopStart()
	if got != 1 {
		t.Errorf("PopStart = %d; want 1", got)
	}
	// Now buffer might look logically like [2,3] with some internal offset

	// Push another element at the end
	b.PushEnd(4)
	// Buffer is now [2,3,4] but wrapped around

	if b.Len() != 3 {
		t.Errorf("Len = %d; want 3", b.Len())
	}

	// Check the order
	expect := []int{2, 3, 4}
	for i, want := range expect {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestCopy(t *testing.T) {
	b := ring.NewBuffer[int](5)
	b.PushEnd(5)
	b.PushEnd(6)
	b.PushEnd(7)

	dst := make([]int, 5)
	n := b.Copy(dst, 0)
	if n != 3 {
		t.Errorf("Copy returned %d; want 3", n)
	}
	if !reflect.DeepEqual(dst[:3], []int{5, 6, 7}) {
		t.Errorf("Copied elements %v; want [5,6,7]", dst[:3])
	}

	// Partial copy starting from an index
	n = b.Copy(dst, 1)
	if n != 2 {
		t.Errorf("Copy from index 1 returned %d; want 2", n)
	}
	if !reflect.DeepEqual(dst[:2], []int{6, 7}) {
		t.Errorf("Copied elements %v; want [6,7]", dst[:2])
	}

	// Copy into smaller slice
	smallDst := make([]int, 2)
	n = b.Copy(smallDst, 0)
	if n != 2 {
		t.Errorf("Copy into smaller slice returned %d; want 2", n)
	}
	if !reflect.DeepEqual(smallDst, []int{5, 6}) {
		t.Errorf("Copied elements %v; want [5,6]", smallDst)
	}

	// Copy from an out-of-range index (should return zero, but not panic)
	outOfRangeDst := make([]int, 3)
	mustPanic(t, func() {
		b.Copy(outOfRangeDst, 5) // index 5 > len (3)
	})
}

func TestDiscardBeyondLength(t *testing.T) {
	b := ring.NewBuffer[int](5)
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)

	discarded := b.DiscardFromStart(10) // More than length
	if discarded != 3 {
		t.Errorf("DiscardFromStart(10) = %d; want 3", discarded)
	}
	if b.Len() != 0 {
		t.Error("Buffer should be empty after discarding all elements.")
	}

	// Refill
	b.PushEnd(4)
	b.PushEnd(5)
	discarded = b.DiscardFromEnd(10) // More than length again
	if discarded != 2 {
		t.Errorf("DiscardFromEnd(10) = %d; want 2", discarded)
	}
	if b.Len() != 0 {
		t.Error("Buffer should be empty after discarding all elements.")
	}
}

func TestIndexOutOfRangePanic(t *testing.T) {
	b := ring.NewBuffer[int](3)
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)

	mustPanic(t, func() { b.Get(-1) })
	mustPanic(t, func() { b.Get(3) }) // valid indices are 0,1,2
	mustPanic(t, func() { b.Get(10) })
}

func TestMixedOperations(t *testing.T) {
	b := ring.NewBuffer[string](4)
	b.PushEnd("A")
	b.PushEnd("B")
	b.PushEnd("C")
	if b.PeekStart() != "A" {
		t.Errorf("PeekStart expected A, got %s", b.PeekStart())
	}
	if b.PeekEnd() != "C" {
		t.Errorf("PeekEnd expected C, got %s", b.PeekEnd())
	}

	got := b.PopStart() // removes "A"
	if got != "A" {
		t.Errorf("PopStart expected A, got %s", got)
	}
	t.Logf("x0 %v", b)
	b.PushStart("Z") // now [Z,B,C]
	t.Logf("x1 %v", b)

	got = b.PopEnd() // removes "C"
	if got != "C" {
		t.Errorf("PopEnd expected C, got %s", got)
	}
	t.Logf("x2 %v", b)
	b.PushEnd("D") // now [Z,B,D]
	t.Logf("x3 %v", b)

	if b.Len() != 3 {
		t.Errorf("Len expected 3, got %d", b.Len())
	}

	expect := []string{"Z", "B", "D"}
	for i, want := range expect {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %s; want %s", i, got, want)
		}
	}
}

func TestGet(t *testing.T) {
	var b ring.Buffer[int]

	for i := 0; i < 100; i++ {
		b.PushEnd(i + 100)
		for j := 0; j < b.Len(); j++ {
			if got, want := b.Get(j), j+100; got != want {
				t.Fatalf("index %d; got %d want %d", j, got, want)
			}
		}
	}
}

func TestPushSliceStart(t *testing.T) {
	b := ring.NewBuffer[int](5)

	// Test basic PushSliceStart
	b.PushSliceStart([]int{1, 2, 3})
	if b.Len() != 3 {
		t.Errorf("Len = %d; want 3", b.Len())
	}

	// Elements should be in order [1, 2, 3]
	for i, want := range []int{1, 2, 3} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}

	// Now push more at the start
	b.PushSliceStart([]int{-1, 0})
	if b.Len() != 5 {
		t.Errorf("Len = %d; want 5", b.Len())
	}

	// Elements should be in order [-1, 0, 1, 2, 3]
	for i, want := range []int{-1, 0, 1, 2, 3} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestPushSliceStartWrapped(t *testing.T) {
	b := ring.NewBuffer[int](4)

	// Fill the buffer
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)

	// Pop one from start to create wrap-around condition
	b.PopStart()

	// Now buffer has [2, 3] with i0 != 0
	// Push a slice at the start that will wrap around
	b.PushSliceStart([]int{10, 11, 12})

	if b.Len() != 5 {
		t.Errorf("Len = %d; want 5", b.Len())
	}

	// Elements should be [10, 11, 12, 2, 3]
	for i, want := range []int{10, 11, 12, 2, 3} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestPushSliceEnd(t *testing.T) {
	b := ring.NewBuffer[int](5)

	// Test basic PushSliceEnd
	b.PushSliceEnd([]int{1, 2, 3})
	if b.Len() != 3 {
		t.Errorf("Len = %d; want 3", b.Len())
	}

	// Elements should be in order [1, 2, 3]
	for i, want := range []int{1, 2, 3} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}

	// Now push more at the end
	b.PushSliceEnd([]int{4, 5})
	if b.Len() != 5 {
		t.Errorf("Len = %d; want 5", b.Len())
	}

	// Elements should be in order [1, 2, 3, 4, 5]
	for i, want := range []int{1, 2, 3, 4, 5} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestPopEndMemoryCleanup(t *testing.T) {
	// This test verifies that PopEnd properly clears the removed element
	// to avoid memory leaks with pointer types
	b := ring.NewBuffer[*int](4)

	// Create some values
	v1, v2, v3 := new(int), new(int), new(int)
	*v1, *v2, *v3 = 1, 2, 3

	b.PushEnd(v1)
	b.PushEnd(v2)
	b.PushEnd(v3)

	// Pop from end
	popped := b.PopEnd()
	if popped != v3 {
		t.Errorf("PopEnd returned wrong value")
	}

	// Get the internal state to verify the element was cleared
	// We'll do this by checking that the buffer doesn't contain v3 anymore
	// after popping it
	found := false
	for v := range b.All() {
		if v == v3 {
			found = true
			break
		}
	}
	if found {
		t.Error("PopEnd did not clear the removed element from visible buffer")
	}
}

func TestPopEndWrapped(t *testing.T) {
	// Test PopEnd when the buffer wraps around
	b := ring.NewBuffer[int](4)

	// Fill buffer: [1, 2, 3, 4]
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)
	b.PushEnd(4)

	// Pop from start to create offset: [2, 3, 4]
	b.PopStart()

	// Push to end, causing wrap: [2, 3, 4, 5]
	b.PushEnd(5)

	// Now i1 should be at position 0 (wrapped)
	// Pop from end should return 5
	popped := b.PopEnd()
	if popped != 5 {
		t.Errorf("PopEnd = %d; want 5", popped)
	}

	if b.Len() != 3 {
		t.Errorf("Len = %d; want 3", b.Len())
	}

	// Verify remaining elements are [2, 3, 4]
	for i, want := range []int{2, 3, 4} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestDiscardFromEndWrapped(t *testing.T) {
	// Test that DiscardFromEnd properly clears only the discarded elements
	// This test creates a scenario where the buffer is wrapped (data at both ends)
	// and DiscardFromEnd needs to only clear elements from the front part
	b := ring.NewBuffer[int](8)

	// Fill buffer completely: [1, 2, 3, 4, 5, 6, 7, 8]
	for i := 0; i < 8; i++ {
		b.PushEnd(i + 1)
	}

	// Remove 5 from start: [6, 7, 8] with i0=5, i1=0 (wrapped)
	b.DiscardFromStart(5)

	// Now push 2 more at the end: [6, 7, 8, 9, 10] with i0=5, i1=2
	b.PushEnd(9)
	b.PushEnd(10)

	// Verify current state
	if b.Len() != 5 {
		t.Errorf("Len before discard = %d; want 5", b.Len())
	}

	// Discard 2 from end: should remove 9, 10
	// After: [6, 7, 8] with i0=5, i1=0
	discarded := b.DiscardFromEnd(2)
	if discarded != 2 {
		t.Errorf("DiscardFromEnd(2) = %d; want 2", discarded)
	}

	// Buffer should now be [6, 7, 8]
	if b.Len() != 3 {
		t.Errorf("Len after discard = %d; want 3", b.Len())
	}

	for i, want := range []int{6, 7, 8} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestSetCap(t *testing.T) {
	b := ring.NewBuffer[int](16)

	// Fill with some elements
	for i := 0; i < 10; i++ {
		b.PushEnd(i)
	}

	initialCap := b.Cap()
	if initialCap < 16 {
		t.Errorf("Initial cap = %d; want >= 16", initialCap)
	}

	// SetCap to a smaller value
	b.SetCap(8)
	if b.Cap() < 8 {
		t.Errorf("Cap after SetCap(8) = %d; want >= 8", b.Cap())
	}

	// Verify data is still intact
	if b.Len() != 10 {
		t.Errorf("Len = %d; want 10", b.Len())
	}
	for i := 0; i < 10; i++ {
		if got := b.Get(i); got != i {
			t.Errorf("Get(%d) = %d; want %d", i, got, i)
		}
	}

	// SetCap to same or smaller shouldn't grow
	oldCap := b.Cap()
	b.SetCap(5) // Less than Len, should stay at current capacity
	if b.Cap() > oldCap {
		t.Errorf("Cap grew unexpectedly: %d -> %d", oldCap, b.Cap())
	}
}

func TestAllIterator(t *testing.T) {
	b := ring.NewBuffer[int](5)
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)

	// Test full iteration
	var collected []int
	for v := range b.All() {
		collected = append(collected, v)
	}
	expected := []int{1, 2, 3}
	if len(collected) != len(expected) {
		t.Errorf("Collected %v; want %v", collected, expected)
	}
	for i, want := range expected {
		if collected[i] != want {
			t.Errorf("collected[%d] = %d; want %d", i, collected[i], want)
		}
	}

	// Test early termination
	collected = nil
	for v := range b.All() {
		collected = append(collected, v)
		if v == 2 {
			break
		}
	}
	if len(collected) != 2 {
		t.Errorf("Early termination: collected %v; want [1, 2]", collected)
	}
}

func TestPushSliceEndWrapped(t *testing.T) {
	b := ring.NewBuffer[int](8)

	// Fill buffer
	for i := 0; i < 8; i++ {
		b.PushEnd(i + 1)
	}

	// Remove from start to create offset
	b.DiscardFromStart(3)

	// Now buffer has [4, 5, 6, 7, 8] with i0=3
	// Push a slice that will wrap around
	b.PushSliceEnd([]int{9, 10, 11})

	if b.Len() != 8 {
		t.Errorf("Len = %d; want 8", b.Len())
	}

	// Should be [4, 5, 6, 7, 8, 9, 10, 11]
	for i, want := range []int{4, 5, 6, 7, 8, 9, 10, 11} {
		if got := b.Get(i); got != want {
			t.Errorf("Get(%d) = %d; want %d", i, got, want)
		}
	}
}

func TestDiscardFromStartWrapped(t *testing.T) {
	b := ring.NewBuffer[int](8)

	// Fill buffer
	for i := 0; i < 8; i++ {
		b.PushEnd(i + 1)
	}

	// Remove from start to wrap
	b.DiscardFromStart(3)

	// Push more to wrap i1
	b.PushEnd(9)
	b.PushEnd(10)

	// Now buffer has [4, 5, 6, 7, 8, 9, 10]
	// Discard more that crosses the wrap boundary
	b.DiscardFromStart(6)

	if b.Len() != 1 {
		t.Errorf("Len = %d; want 1", b.Len())
	}

	if got := b.Get(0); got != 10 {
		t.Errorf("Get(0) = %d; want 10", got)
	}
}

func TestDiscardFromEndCrossWrap(t *testing.T) {
	// Test the else branch of DiscardFromEnd (when i1-n < 0)
	b := ring.NewBuffer[int](8)

	// Fill buffer completely
	for i := 0; i < 8; i++ {
		b.PushEnd(i + 1)
	}

	// Remove from start to create wrap
	b.DiscardFromStart(6)

	// Now buffer has [7, 8] with i0=6, i1=0 (wrapped)
	// Push more elements
	b.PushEnd(9)
	b.PushEnd(10)
	b.PushEnd(11)

	// Now buffer is [7, 8, 9, 10, 11] with i0=6, i1=3
	// Discard 4 from end (which will need to clear across the wrap)
	b.DiscardFromEnd(4)

	if b.Len() != 1 {
		t.Errorf("Len = %d; want 1", b.Len())
	}

	if got := b.Get(0); got != 7 {
		t.Errorf("Get(0) = %d; want 7", got)
	}
}

func TestCopyFromEnd(t *testing.T) {
	b := ring.NewBuffer[int](5)
	b.PushEnd(1)
	b.PushEnd(2)
	b.PushEnd(3)
	b.PushEnd(4)

	// Copy from the end (at boundary of buffer)
	dst := make([]int, 2)
	n := b.Copy(dst, b.Len())
	if n != 0 {
		t.Errorf("Copy from b.Len() returned %d; want 0", n)
	}
}

func TestZeroBuffer(t *testing.T) {
	// Test zero-value buffer
	var b ring.Buffer[string]

	if b.Len() != 0 {
		t.Errorf("Zero buffer Len = %d; want 0", b.Len())
	}

	if b.Cap() != 0 {
		t.Errorf("Zero buffer Cap = %d; want 0", b.Cap())
	}

	// Should be able to push to zero buffer
	b.PushEnd("hello")
	if b.Len() != 1 {
		t.Errorf("Len after push = %d; want 1", b.Len())
	}

	if got := b.PeekStart(); got != "hello" {
		t.Errorf("PeekStart = %s; want hello", got)
	}
}

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, but code did not panic")
		}
	}()
	f()
}
