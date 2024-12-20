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
	t.Logf("%#v", b)
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
	t.Logf("x0 %#v", b)
	b.PushStart("Z") // now [Z,B,C]
	t.Logf("x1 %#v", b)

	got = b.PopEnd() // removes "C"
	if got != "C" {
		t.Errorf("PopEnd expected C, got %s", got)
	}
	t.Logf("x2 %#v", b)
	b.PushEnd("D") // now [Z,B,D]
	t.Logf("x3 %#v", b)

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

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, but code did not panic")
		}
	}()
	f()
}
