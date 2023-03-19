package watcher

import "sync"

type Updater[T any] interface {
	Update(*T, T) bool
}

// TODO
// type Value[T any] = ValueU[T, Always[T]]

// Value represents a shared value that can be watched for changes. Methods on
// a Value may be called concurrently.
//
// The zero Value is ok to use; watchers on the zero value
// will block until Set is called.
//
// U is used to update the value. U.Update must be callable on
// the zero value of U.
type Value[T any, U Updater[T]] struct {
	wait sync.Cond
	// mu guards the fields below it.
	mu      sync.RWMutex
	val     T
	version int
	closed  bool
}

// NewValue creates a new Value holding the given initial value.
func NewValue[T any, U Updater[T]](initial T) *Value[T, U] {
	var v Value[T, U]
	v.Set(initial)
	return &v
}

func (v *Value[T, U]) needsInit() bool {
	return v.wait.L == nil
}

func (v *Value[T, U]) init() {
	if v.needsInit() {
		v.wait.L = v.mu.RLocker()
	}
}

// Set sets the shared value to val.
func (v *Value[T, U]) Set(val T) {
	v.mu.Lock()
	v.init()
	if (*new(U)).Update(&v.val, val) {
		v.version++
	}
	v.mu.Unlock()
	v.wait.Broadcast()
}

// Close closes the Value, unblocking any outstanding watchers.  Close always
// returns nil.
func (v *Value[T, U]) Close() error {
	v.mu.Lock()
	v.init()
	v.closed = true
	v.val = *new(T)
	v.mu.Unlock()
	v.wait.Broadcast()
	return nil
}

// Closed reports whether the value has been closed.
func (v *Value[T, U]) Closed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.closed
}

// Get returns the current value.
// If the watcher has been closed, it returns the zero value.
func (v *Value[T, U]) Get() T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.val
}

// GetOK returns the most recently set value and reports whether
// it is valid. After v has been closed, GetOK will always return
// *new(T), false.
func (v *Value[T, U]) GetOK() (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.val, v.closed
}

// Watch returns a Watcher that can be used to watch for changes to the value.
func (v *Value[T, U]) Watch() Watcher[T] {
	return &watcher[T, U]{value: v}
}

// Watcher represents a single watcher of a shared value.
type Watcher[T any] interface {
	Next() bool
	Close()
	Value() T
}

// Watcher represents a single watcher of a shared value.
type watcher[T any, U Updater[T]] struct {
	value   *Value[T, U]
	version int
	current T
	closed  bool
}

// Next blocks until there is a new value to be retrieved from the value that is
// being watched. It also unblocks when the value or the Watcher itself is
// closed. Next returns false if the value or the Watcher itself have been
// closed.
func (w *watcher[T, U]) Next() bool {
	val := w.value
	val.mu.RLock()
	defer val.mu.RUnlock()
	if val.needsInit() {
		val.mu.RUnlock()
		val.mu.Lock()
		val.init()
		val.mu.Unlock()
		val.mu.RLock()
	}

	// We can go around this loop a maximum of two times,
	// because the only thing that can cause a Wait to
	// return is for the condition to be triggered,
	// which can only happen if the value is set (causing
	// the version to increment) or it is closed
	// causing the closed flag to be set.
	// Both these cases will cause Next to return.
	for {
		if w.version != val.version {
			if (*new(U)).Update(&w.current, val.val) {
				w.version = val.version
				return true
			}
		}
		if val.closed || w.closed {
			return false
		}

		// Wait releases the lock until triggered and then reacquires the lock.
		val.wait.Wait()
	}
}

// Close closes the Watcher without closing the underlying
// value. It may be called concurrently with Next.
func (w *watcher[T, U]) Close() {
	w.value.mu.Lock()
	w.value.init()
	w.closed = true
	w.current = *new(T)
	w.value.mu.Unlock()
	w.value.wait.Broadcast()
}

// Value returns the last value that was retrieved from the watched Value by
// Next.
func (w *watcher[T, U]) Value() T {
	return w.current
}

// Always implements Updater by always updating the value.
type Always[T any] struct{}

func (Always[T]) Update(dst *T, src T) bool {
	*dst = src
	return true
}

type IfUnequal[T comparable] struct{}

func (u IfUnequal[T]) Update(old *T, new T) bool {
	if *old == new {
		return false
	}
	*old = new
	return true
}
