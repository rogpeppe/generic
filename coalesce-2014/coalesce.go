package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Merger[T any] interface {
	Merge(T) T
}

func coalesce[Event Merger[Event]](in <-chan Event, out chan<- Event) {
	haveEvent := false
	var event Event
	timer := time.NewTimer(0)

	var timerCh <-chan time.Time
	var outCh chan<- Event

	for {
		select {
		case e := <-in:
			if haveEvent {
				event = event.Merge(e)
			} else {
				event = e
			}
			haveEvent = true
			if timerCh == nil {
				timer.Reset(500 * time.Millisecond)
				timerCh = timer.C
			}
		case <-timerCh:
			outCh = out
			timerCh = nil
		case outCh <- event:
			haveEvent = false
			outCh = nil
		}
	}
}

type Event int

func (e Event) Merge(other Event) Event { return e + other }

func slowReceive(in <-chan Event) {
	for i := 0; i < 10; i++ {
		time.Sleep(1500 * time.Millisecond)
		fmt.Println("Received:", <-in)
	}
}

func produce(out chan<- Event) {
	for {
		delay := time.Duration(rand.Intn(5)+1) * time.Second
		nMessages := rand.Intn(10) + 1
		time.Sleep(delay)
		for i := 0; i < nMessages; i++ {
			e := Event(rand.Intn(10))
			fmt.Println("Producing:", e)
			out <- e
		}
	}
}

func main() {
	source := make(chan Event)
	output := make(chan Event)

	go produce(source)
	go coalesce(source, output)
	slowReceive(output)
}
