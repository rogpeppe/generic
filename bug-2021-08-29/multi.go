// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package genericio

type eofReader[T any] struct{}

func (eofReader[T]) Read([]T) (int, error) {
	return 0, EOF
}

type multiReader[T any] struct {
	readers []Reader[T]
}

func (mr *multiReader[T]) Read(p []T) (n int, err error) {
	for len(mr.readers) > 0 {
		// Optimization to flatten nested multiReaders (Issue 13558).
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*multiReader[T]); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF {
			// Use eofReader instead of nil to avoid nil panic
			// after performing flatten (Issue 18232).
			mr.readers[0] = eofReader[T]{} // permit earlier GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF {
			if err == EOF && len(mr.readers) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

// MultiReader returns a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.  If any of the readers
// return a non-nil, non-EOF error, Read will return that error.
func MultiReader[T any](readers ...Reader[T]) Reader[T] {
	r := make([]Reader[T], len(readers))
	copy(r, readers)
	return &multiReader[T]{r}
}

type multiWriter[T any] struct {
	writers []Writer[T]
}

func (t *multiWriter[T]) Write(p []T) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Each write is written to each listed writer, one at a time.
// If a listed writer returns an error, that overall write operation
// stops and returns the error; it does not continue down the list.
func MultiWriter[T any](writers ...Writer[T]) Writer[T] {
	allWriters := make([]Writer[T], 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter[T]); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter[T]{allWriters}
}
