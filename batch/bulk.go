// Package batch provides support for batching up singular calls into
// multiple-value calls which may be more efficient.
//
// Eventually, this could potentially live in x/sync alongside
// the singleflight package.
package batch

import (
	"fmt"
	"sync"
	"time"
)

// Caller represents a batch caller. It accumulates multiple calls
// into a single "batch" call to reduce the total number of calls made.
// The zero value is equivalent to NewCaller(0, 0).
type Caller[Value, Result any] struct {
	initialDelay   time.Duration
	maxConcurrency int
	mu             sync.Mutex
	sem            chan struct{}
	acc            *accumulator[Value, Result]
}

// NewCaller returns a Caller that issues a maximum of maxConcurrency
// concurrent calls, and delays for at least initialDelay after issuing a
// call to accumulate possible extra calls to avoid the first call being issued
// immediately.
//
// If maxConcurrency is non-positive, 1 concurrent call will be allowed.
func NewCaller[Value, Result any](maxConcurrency int, initialDelay time.Duration) *Caller[Value, Result] {
	return &Caller[Value, Result]{
		initialDelay:   initialDelay,
		maxConcurrency: maxConcurrency,
	}
}

// DoChan is like Do but returns a channel on which the result can be
// received instead of the result itself.
func (g *Caller[V, R]) DoChan(v V, call func(vs ...V) ([]R, error)) <-chan Result[R] {
	// TODO if we changed the call function signature so that the
	// result slice was passed in rather than the other way around,
	// we'd be able to use sync.Pool for result slice allocations.
	g.mu.Lock()
	if g.sem == nil {
		n := g.maxConcurrency
		if n <= 0 {
			n = 1
		}
		g.sem = make(chan struct{}, n)
	}
	acc := g.acc
	isInitial := acc == nil
	if isInitial {
		acc = new(accumulator[V, R])
		g.acc = acc
	}
	acc.args = append(acc.args, v)
	resultc := make(chan Result[R], 1)
	acc.results = append(acc.results, resultc)
	g.mu.Unlock()

	if isInitial {
		g.doCall(call)
	}
	return resultc
}

// Do does the equivalent of:
//
//	rs, err := call(v)
//	if err != nil {
//		return _, err
//	}
//	return rs[0], nil
//
// except that only a limited number of Do methods can
// be excecuting at a time. If the maximum count has been
// reached, additional Do calls will accumulate argument values into
// a slice and use the same call function, which should return
// a slice with the results in corresponding elements to the arguments.
//
func (g *Caller[V, R]) Do(v V, call func(vs ...V) ([]R, error)) (R, error) {
	r := <-g.DoChan(v, call)
	return r.Val, r.Err
}

// Result represents the result of a call.
type Result[R any] struct {
	Val R
	Err error
}

// accumulator is used to accumulate arguments and result channels
// prior to a call.
type accumulator[V, R any] struct {
	args    []V
	results []chan<- Result[R]
}

func (g *Caller[V, R]) doCall(fn func(...V) ([]R, error)) {
	if g.initialDelay > 0 {
		time.Sleep(g.initialDelay)
	}
	// Wait until a call slot is available. Any calls that happen
	// in the meantime will add their arguments to g.acc
	// and we'll use them when we make the call.
	g.sem <- struct{}{}
	defer func() {
		<-g.sem
	}()
	// Remove this call from the group. We're about
	// to start executing it.
	g.mu.Lock()
	acc := g.acc
	g.acc = nil
	g.mu.Unlock()

	rs, err := fn(acc.args...)
	if err == nil && len(rs) != len(acc.args) {
		err = fmt.Errorf("unexpected result slice length (got %d want %d)", len(rs), len(acc.args))
	}
	if err != nil {
		for _, r := range acc.results {
			r <- Result[R]{
				Err: err,
			}
		}
		return
	}
	for i, r := range acc.results {
		r <- Result[R]{
			Val: rs[i],
		}
	}
}
