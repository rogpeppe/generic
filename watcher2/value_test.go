package watcher

import (
	"fmt"
	"time"

	"testing"

	qt "github.com/frankban/quicktest"
)

func ExampleWatcher_Next() {
	var v Value[string, Always[string]]

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
	c := qt.New(t)
	var v Value[string, Always[string]]
	expected := "12345"
	v.Set(expected)
	got := v.Get()
	c.Assert(got, qt.Equals, expected)
	c.Assert(v.Closed(), qt.IsFalse)
}

func TestValueInitial(t *testing.T) {
	c := qt.New(t)
	expected := "12345"
	v := NewValue[string, Always[string]](expected)
	got := v.Get()
	c.Assert(got, qt.Equals, expected)
	c.Assert(v.Closed(), qt.IsFalse)
}

func TestValueClose(t *testing.T) {
	c := qt.New(t)
	expected := "12345"
	v := NewValue[string, Always[string]](expected)
	c.Assert(v.Close(), qt.IsNil)

	isClosed := v.Closed()
	c.Assert(isClosed, qt.IsTrue)
	got := v.Get()
	c.Assert(got, qt.Equals, "")

	// test that we can close multiple times without a problem
	c.Assert(v.Close(), qt.IsNil)
}

func TestWatcher(t *testing.T) {
	c := qt.New(t)
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string, Always[string]]

	go func() {
		for _, s := range vals {
			v.Set(s)
			ch <- true
		}
		v.Close()
	}()

	w := v.Watch()
	c.Assert(w.Next(), qt.IsTrue)
	c.Assert(w.Value(), qt.Equals, vals[0])

	// test that we can get the same value multiple times
	c.Assert(w.Value(), qt.Equals, vals[0])
	<-ch

	// now try skipping a value by calling next without getting the value
	c.Assert(w.Next(), qt.IsTrue)
	<-ch

	c.Assert(w.Next(), qt.IsTrue)
	c.Assert(w.Value(), qt.Equals, vals[2])
	<-ch

	c.Assert(w.Next(), qt.IsFalse)
}

func TestDoubleSet(t *testing.T) {
	c := qt.New(t)
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string, Always[string]]

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
	c.Assert(w.Next(), qt.IsTrue)
	c.Assert(w.Value(), qt.Equals, vals[0])
	<-ch

	// since we did two sets before sending on the channel,
	// we should just get vals[2] here and not get vals[1]
	c.Assert(w.Next(), qt.IsTrue)
	c.Assert(w.Value(), qt.Equals, vals[2])
}

func TestTwoReceivers(t *testing.T) {
	c := qt.New(t)
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string, Always[string]]

	watcher := func() {
		w := v.Watch()
		x := 0
		for w.Next() {
			c.Assert(w.Value(), qt.Equals, vals[x])
			x++
			<-ch
		}
		c.Assert(x, qt.Equals, len(vals))
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
	c := qt.New(t)
	vals := []string{"one", "two", "three"}

	// blocking on the channel forces the scheduler to let the other goroutine
	// run for a bit, so we get predictable results.  This is not necessary for
	// normal use of the watcher.
	ch := make(chan bool)

	var v Value[string, Always[string]]

	w := v.Watch()
	go func() {
		x := 0
		for w.Next() {
			c.Assert(w.Value(), qt.Equals, vals[x])
			x++
			<-ch
		}
		// the value will only get set once before the watcher is closed
		c.Assert(x, qt.Equals, 1)
		<-ch
	}()

	v.Set(vals[0])
	ch <- true
	w.Close()
	ch <- true

	// prove the value is not closed, even though the watcher is
	c.Assert(v.Closed(), qt.IsFalse)
}

func TestWatchZeroValue(t *testing.T) {
	c := qt.New(t)
	var v Value[struct{}, Always[struct{}]]
	ch := make(chan bool)
	go func() {
		w := v.Watch()
		ch <- true
		ch <- w.Next()
	}()
	<-ch
	v.Set(struct{}{})
	c.Assert(<-ch, qt.IsTrue)
}

func TestUpdateIfUnequal(t *testing.T) {
	c := qt.New(t)
	var v Value[string, IfUnequal[string]]
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
	c.Assert(got, qt.DeepEquals, []string{"first", "second"})
}
