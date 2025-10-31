package merge

import (
	"iter"
	"slices"
	"testing"

	"github.com/go-quicktest/qt"
)

func TestMerge(t *testing.T) {
	qt.Assert(t, qt.DeepEquals(
		slices.Collect(Merge(slices.Values([]int{1, 2, 5, 7, 43, 87}), slices.Values([]int{1, 3, 6, 7, 9}))),
		[]int{1, 2, 3, 5, 6, 7, 9, 43, 87},
	))

	qt.Assert(t, qt.DeepEquals(
		slices.Collect(MergeMulti(
			slices.Values([]int{4, 6, 7}),
			slices.Values([]int{}),
			slices.Values([]int{2, 6, 77, 87}),
			slices.Values([]int{1, 65, 99}),
		)),
		[]int{1, 2, 4, 6, 7, 65, 77, 87, 99},
	))
}

func BenchmarkMerge(b *testing.B) {
	it := MergeMulti(randIter(0), randIter(1))
	prev := int64(-1)
	i := 0
	for x := range it {
		if i >= b.N {
			break
		}
		if x < prev {
			b.Fatalf("unordered")
		}
		prev = x
		i++
	}
}

func BenchmarkMergeSetup(b *testing.B) {
	for b.Loop() {
		for range Merge(randIter(0), randIter(1)) {
			break
		}
	}
}

func randIter(seed int64) iter.Seq[int64] {
	return func(yield func(int64) bool) {
		//r := rand.New(rand.NewSource(seed))
		x := int64(seed)
		for {
			x += 10
			//x += r.Int63n(10) + 1
			if !yield(x) {
				return
			}
		}
	}
}
