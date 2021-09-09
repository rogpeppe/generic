package poller

import (
	"testing"
	"time"
)

// WaitFor continuously calls poll until check returns true. It then polls for
// a little longer to make sure that poll still returns a value v such that check(v)
// is true. If the condition never happens, or the condition becomes true
// and then false, it invokes t.Fatal.
//
// If poll returns an error, WaitFor calls Fatal.
//
// WaitFor returns the last value that poll returned.
func WaitFor[T any](t *testing.T, timeout time.Duration, poll func() (T, error), check func(T) bool) T {
	panic("unimplemented")
}
