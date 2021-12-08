package slice

import "testing"

var compareTests = []struct {
	s1, s2 []string
	want   int
}{{
	s1:   nil,
	s2:   nil,
	want: 0,
}, {
	s1:   []string{"a"},
	s2:   nil,
	want: 1,
}, {
	s1:   nil,
	s2:   []string{"a"},
	want: -1,
}, {
	s1:   []string{"a"},
	s2:   []string{"a"},
	want: 0,
}, {
	s1:   []string{"b"},
	s2:   []string{"a"},
	want: 1,
}, {
	s1:   []string{"a"},
	s2:   []string{"b"},
	want: -1,
}, {
	s1:   []string{"a", "b"},
	s2:   []string{"a"},
	want: 1,
}, {
	s1:   []string{"a"},
	s2:   []string{"a", "b"},
	want: -1,
}}

func TestCompare(t *testing.T) {
	for _, test := range compareTests {
		t.Run("", func(t *testing.T) {
			got := Compare(test.s1, test.s2)
			if got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}
