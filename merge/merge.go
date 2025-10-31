package merge

import (
	"cmp"
	"fmt"
	"iter"
)

// Join implements a join function that just returns whatever
// value is being joined.
func Join[T comparable](x0 T, has0 bool, x1 T, has1 bool) T {
	if has0 {
		return x0
	}
	return x1
}

// Merge implement stream merging on a set of ordered values.
func Merge[T cmp.Ordered](it0, it1 iter.Seq[T]) iter.Seq[T] {
	return MergeGeneral(it0, it1, cmp.Compare[T], Join[T])
}

func MergeMulti[T cmp.Ordered](its ...iter.Seq[T]) iter.Seq[T] {
	return MergeMultiGeneral(cmp.Compare[T], Join[T], its...)
}

func MergeMultiGeneral[T any](cmp func(T, T) int, join func(T, bool, T, bool) T, its ...iter.Seq[T]) iter.Seq[T] {
	if len(its) == 0 {
		return func(yield func(T) bool) {}
	}
	r := its[0]
	for _, it := range its[1:] {
		r = MergeGeneral(r, it, cmp, join)
	}
	return r
}

func MergeGeneral[T0, T1 any](it0, it1 iter.Seq[T0], cmp func(T0, T0) int, join func(T0, bool, T0, bool) T1) iter.Seq[T1] {
	return func(yield func(T1) bool) {
		next0, stop0 := iter.Pull(it0)
		defer stop0()
		next1, stop1 := iter.Pull(it1)
		defer stop1()
		var x0, x1 T0
		has0, has1 := false, false
		first0, first1 := true, true
		for {
			if !has0 && next0 != nil {
				if n0, ok := next0(); ok {
					has0 = true
					if !first0 && cmp(x0, n0) >= 0 {
						panic(fmt.Errorf("out of order item in sequence (%v <= %v)", x0, n0))
					}
					x0 = n0
					first0 = false
				} else {
					next0 = nil
				}
			}
			if !has1 && next1 != nil {
				if n1, ok := next1(); ok {
					has1 = true
					if !first1 && cmp(x1, n1) >= 0 {
						panic("out of order item in sequence")
					}
					x1 = n1
					first1 = false
				} else {
					next1 = nil
				}
			}
			switch {
			case has0 && has1:
				c := cmp(x0, x1)
				switch {
				case c < 0:
					if !yield(join(x0, true, *new(T0), false)) {
						return
					}
					has0 = false
				case c > 0:
					if !yield(join(*new(T0), false, x1, true)) {
						return
					}
					has1 = false
				default:
					if !yield(join(x0, true, x1, true)) {
						return
					}
					has0, has1 = false, false
				}
			case has0:
				if !yield(join(x0, true, *new(T0), false)) {
					return
				}
				has0 = false
			case has1:
				if !yield(join(*new(T0), false, x1, true)) {
					return
				}
				has1 = false
			default:
				return
			}
		}
	}
}
