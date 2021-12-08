package batch

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

func TestSingleCall(t *testing.T) {
	var caller Caller[int, string]
	s, err := caller.Do(123, func(is ...int) ([]string, error) {
		if got, want := len(is), 1; got != want {
			t.Errorf("unexpected argument count; got %d want %d", got, want)
		}
		return []string{fmt.Sprint(is[0])}, nil
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if got, want := s, "123"; got != want {
		t.Errorf("unexpected result; got %#v want %#v", got, want)
	}
}

func TestMultipleCalls(t *testing.T) {
	caller := NewCaller[int, string](2, 0)

	callDuration := 50 * time.Millisecond
	stringer := func(is ...int) ([]string, error) {
		time.Sleep(callDuration)
		r := make([]string, len(is))
		for i, v := range is {
			r[i] = fmt.Sprint(v)
		}
		return r, nil
	}

	t0 := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := caller.Do(i, stringer)
			if err != nil {
				t.Errorf("got error from Do: %v", err)
			}
			if got, want := r, fmt.Sprint(i); got != want {
				t.Errorf("unexpected result; got %q want %q", got, want)
			}
		}()
	}
	wg.Wait()
	total := time.Since(t0)
	if got, want := total, 2*callDuration+10*time.Millisecond; got > want {
		t.Errorf("total took too long; got %v want %v", got, want)
	}
	log.Printf("total time %v", total)
}
