package watcher

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
)

func ExampleWatcher_Next() {
	var v Value[string]

	// The channel is not necessary for normal use of the watcher.
	// It just makes the test output predictable.
	ch := make(chan bool)

	go func() {
		for x := 0; x < 3; x++ {
			v.Set(fmt.Sprintf("value%d", x))
			ch <- true
		}
		v.Close()
	}()
	w := v.Watch()
	for w.Next() {
		fmt.Println(w.Value())
		<-ch
	}

	// output:
	// value0
	// value1
	// value2
}

func TestValueGetSet(t *testing.T) {
	var v Value[string]
	expected := "12345"
	v.Set(expected)
	got := v.Get()
	qt.Assert(t, qt.Equals(got, expected))
	qt.Assert(t, qt.IsFalse(v.Closed()))
}

func TestValueInitial(t *testing.T) {
	expected := "12345"
	v := NewValue(expected)
	got := v.Get()
	qt.Assert(t, qt.Equals(got, expected))
	qt.Assert(t, qt.IsFalse(v.Closed()))
}

func TestValueClose(t *testing.T) {
	expected := "12345"
	v := NewValue(expected)
	qt.Assert(t, qt.IsNil(v.Close()))

	isClosed := v.Closed()
	qt.Assert(t, qt.IsTrue(isClosed))
	got := v.Get()
	qt.Assert(t, qt.Equals(got, ""))

	// test that we can close multiple times without a problem
	qt.Assert(t, qt.IsNil(v.Close()))
}

func TestWatcher(t *testing.T) {
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string]

	go func() {
		for _, s := range vals {
			v.Set(s)
			ch <- true
		}
		v.Close()
	}()

	w := v.Watch()
	qt.Assert(t, qt.IsTrue(w.Next()))
	qt.Assert(t, qt.Equals(w.Value(), vals[0]))

	// test that we can get the same value multiple times
	qt.Assert(t, qt.Equals(w.Value(), vals[0]))
	<-ch

	// now try skipping a value by calling next without getting the value
	qt.Assert(t, qt.IsTrue(w.Next()))
	<-ch

	qt.Assert(t, qt.IsTrue(w.Next()))
	qt.Assert(t, qt.Equals(w.Value(), vals[2]))
	<-ch

	qt.Assert(t, qt.IsFalse(w.Next()))
}

func TestDoubleSet(t *testing.T) {
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string]

	go func() {
		v.Set(vals[0])
		ch <- true
		v.Set(vals[1])
		v.Set(vals[2])
		ch <- true
		v.Close()
		ch <- true
	}()

	w := v.Watch()
	qt.Assert(t, qt.IsTrue(w.Next()))
	qt.Assert(t, qt.Equals(w.Value(), vals[0]))
	<-ch

	// since we did two sets before sending on the channel,
	// we should just get vals[2] here and not get vals[1]
	qt.Assert(t, qt.IsTrue(w.Next()))
	qt.Assert(t, qt.Equals(w.Value(), vals[2]))
}

func TestTwoReceivers(t *testing.T) {
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string]

	watcher := func() {
		w := v.Watch()
		x := 0
		for w.Next() {
			qt.Assert(t, qt.Equals(w.Value(), vals[x]))
			x++
			<-ch
		}
		qt.Assert(t, qt.Equals(x, len(vals)))
		<-ch
	}

	go watcher()
	go watcher()

	for _, val := range vals {
		v.Set(val)
		ch <- true
		ch <- true
	}

	v.Close()
	ch <- true
	ch <- true
}

func TestCloseWatcher(t *testing.T) {
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string]

	w := v.Watch()
	go func() {
		x := 0
		for w.Next() {
			qt.Assert(t, qt.Equals(w.Value(), vals[x]))
			x++
			<-ch
		}
		// the value will only get set once before the watcher is closed
		qt.Assert(t, qt.Equals(x, 1))
		<-ch
	}()

	v.Set(vals[0])
	ch <- true
	w.Close()
	ch <- true

	// prove the value is not closed, even though the watcher is
	qt.Assert(t, qt.IsFalse(v.Closed()))
}

func TestWatchZeroValue(t *testing.T) {
	var v Value[struct{}]
	ch := make(chan bool)
	go func() {
		w := v.Watch()
		ch <- true
		ch <- w.Next()
	}()
	<-ch
	v.Set(struct{}{})
	qt.Assert(t, qt.IsTrue(<-ch))
}

func TestUpdateIfUnequal(t *testing.T) {
	v := WithUpdater[string](IfUnequal[string])
	go func() {
		v.Set("first")
		time.Sleep(time.Millisecond)
		v.Set("first")
		time.Sleep(time.Millisecond)
		v.Set("second")
		time.Sleep(time.Millisecond)
		v.Close()
	}()
	var got []string
	for w := v.Watch(); w.Next(); {
		got = append(got, w.Value())
	}
	qt.Assert(t, qt.DeepEquals(got, []string{"first", "second"}))
}
