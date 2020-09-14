// Code generated by go2go; DO NOT EDIT.


//line ctrie.go2:24
package ctrie

//line ctrie.go2:24
import (
//line ctrie.go2:24
 "bytes"
//line ctrie.go2:24
 "fmt"
//line ctrie.go2:24
 "github.com/rogpeppe/generic/gatomic"
//line ctrie.go2:24
 "hash/maphash"
//line ctrie.go2:24
 "math/bits"
//line ctrie.go2:24
 "strconv"
//line ctrie.go2:24
 "sync"
//line ctrie.go2:24
 "sync/atomic"
//line ctrie.go2:24
 "testing"
//line ctrie.go2:24
 "time"
//line ctrie.go2:24
 "unsafe"
//line ctrie.go2:24
)

//line ctrie.go2:35
const (
//line ctrie.go2:37
 w = 5

//line ctrie.go2:40
 exp2 = 32
)

var seed = maphash.MakeSeed()

func StringHash(key string) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	h.WriteString(key)
	return h.Sum64()
}

func BytesHash(key []byte) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	h.Write(key)
	return h.Sum64()
}

type String string

func (s String) Hash() uint64 {
	return StringHash(string(s))
}

//line ctrie.go2:78
type generation struct{ _ bool }

//line ctrie.go2:792
type branch interface{}

//line ctrie.go2:895
func flagPos(hashcode uint32, lev uint, bmp uint32) (uint32, int) {
	idx := (hashcode >> lev) & 0x1f
	flag := uint32(1) << idx
	pos := bits.OnesCount32(bmp & (flag - 1))
	return flag, pos
}

//line ctrie.go2:900
type Importable୦ int
//line ctrie.go2:900
type _ bytes.Buffer

//line ctrie.go2:900
var _ = fmt.Errorf

//line ctrie.go2:900
type _ gatomic.Importable୦
//line ctrie.go2:900
type _ maphash.Hash

//line ctrie.go2:900
var _ = bits.Add
//line ctrie.go2:900
var _ = strconv.AppendBool

//line ctrie.go2:900
type _ sync.Cond

//line ctrie.go2:900
var _ = atomic.AddInt32
//line ctrie.go2:900
var _ = testing.AllocsPerRun

//line ctrie.go2:900
const _ = time.ANSIC

//line ctrie.go2:900
type _ unsafe.Pointer