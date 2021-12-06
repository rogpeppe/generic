package watcher

import "sync"

// Value represents a shared value that can be watched for changes. Methods on
// a Value may be called concurrently.
//
// The zero Value is ok to use; watchers on the zero value
// will block until Set is called.
//
// By default, watchers will be notified whenever Set is called,
// but WithUpdater can be used to trigger notifications less often.
type Value[T any] struct {
	wait   sync.Cond
	update UpdateFunc[T]
	// mu guards the fields below it.
	mu      sync.RWMutex
	val     T
	version int
	closed  bool
}

// NewValue creates a new Value holding the given initial value.
func NewValue[T any](initial T) *Value[T] {
	var v Value[T]
	v.Set(initial)
	return &v
}

// WithUpdater returns a value that uses the given function
// to update the Value and report whether it's changed.
//
// The default updater is Always.
func WithUpdater[T any](updater UpdateFunc[T]) *Value[T] {
	return &Value[T]{
		update: updater,
	}
}

func (v *Value[T]) needsInit() bool {
	return v.wait.L == nil
}

func (v *Value[T]) init() {
	if v.needsInit() {
		v.wait.L = v.mu.RLocker()
		if v.update == nil {
			v.update = Always[T]
		}
	}
}

// Set sets the shared value to val.
func (v *Value[T]) Set(val T) {
	v.mu.Lock()
	v.init()
	if v.update(&v.val, val) {
		v.version++
	}
	v.mu.Unlock()
	v.wait.Broadcast()
}

// Close closes the Value, unblocking any outstanding watchers.  Close always
// returns nil.
func (v *Value[T]) Close() error {
	v.mu.Lock()
	v.init()
	v.closed = true
	v.val = *new(T)
	v.mu.Unlock()
	v.wait.Broadcast()
	return nil
}

// Closed reports whether the value has been closed.
func (v *Value[T]) Closed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.closed
}

// Get returns the current value.
// If the watcher has been closed, it returns the zero value.
func (v *Value[T]) Get() T {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.val
}

// GetOK returns the most recently set value and reports whether
// it is valid. After v has been closed, GetOK will always return
// *new(T), false.
func (v *Value[T]) GetOK() (T, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.val, v.closed
}

// Watch returns a Watcher that can be used to watch for changes to the value.
func (v *Value[T]) Watch() *Watcher[T] {
	return &Watcher[T]{value: v}
}

// Watcher represents a single watcher of a shared value.
type Watcher[T any] struct {
	value   *Value[T]
	version int
	current T
	closed  bool
}

// Next blocks until there is a new value to be retrieved from the value that is
// being watched. It also unblocks when the value or the Watcher itself is
// closed. Next returns false if the value or the Watcher itself have been
// closed.
func (w *Watcher[T]) Next() bool {
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
			if val.update(&w.current, val.val) {
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
func (w *Watcher[T]) Close() {
	w.value.mu.Lock()
	w.value.init()
	w.closed = true
	w.current = *new(T)
	w.value.mu.Unlock()
	w.value.wait.Broadcast()
}

// Value returns the last value that was retrieved from the watched Value by
// Next.
func (w *Watcher[T]) Value() T {
	return w.current
}

// UpdateFunc is the type of a function used to update
// a value. It should update old to be the same as new
// and report whether old has changed.
type UpdateFunc[T any] func(old *T, new T) bool

// Always is the default updater function. It just
// assigns new to old and returns true.
func Always[T any](old *T, new T) bool {
	*old = new
	return true
}

// IfUnequal reports a value as having changed only
// if new != *old.
func IfUnequal[T comparable](old *T, new T) bool {
	if *old == new {
		return false
	}
	*old = new
	return true
}
