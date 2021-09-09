package gatomic

import (
	"testing"
)

func TestLoadStorePointer(t *testing.T) {
	var x int
	var p *int
	StorePointer(&p, &x)
	pp := LoadPointer(&p)
	*pp = 12
	if x != 12 {
		t.Fatal("unexpected value")
	}
}
