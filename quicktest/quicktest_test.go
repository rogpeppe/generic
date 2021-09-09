package quicktest

import "testing"

func TestFoo(t *testing.T) {
	x := 5
	Assert(t, x, Equals(5))
}
