package set

import (
	"fmt"
	"strings"
	"testing"
)

func TestIntSet(t *testing.T) {
	t.Skip("needs more implementation!")
	s1 := &BitSet{}
	s1.Add(99)
	t.Logf(toString(s1))
}

type intSet[T Set[T, Int]] Set[T, Int]

// show prints all the integers in the given set.
func toString[S intSet[S]](set S) string {
	nset := set.New()
	for _, x := range []Int{4, 6, 3, 6} {
		nset.Add(x)
	}
	set.Union(set, nset)
	var buf strings.Builder
	for iter := set.Iter(); iter.Next(); {
		fmt.Fprintf(&buf, "%d\n", iter.Item())
	}
	return buf.String()
}
