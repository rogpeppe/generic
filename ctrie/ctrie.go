/*
Copyright 2015 Workiva, LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package ctrie provides an implementation of the Map data structure, which is
a concurrent, lock-free hash trie. This data structure was originally presented
in the paper Concurrent Tries with Efficient Non-Blocking Clones:

https://axel22.github.io/resources/docs/ctries-clone.pdf
*/
package ctrie

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"math/bits"

	"github.com/rogpeppe/generic/gatomic"
)

const (
	// w controls the number of branches at a node (2^w branches).
	w = 5

	// exp2 is 2^w, which is the hashcode space.
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

// Map implements a map that can be updated concurrently
// and also has a low cost snapshot operation.
type Map[Key, Value any] struct {
	root     *iNode[Key, Value]
	readOnly bool
	hashFunc func(Key) uint64
	eqFunc   func(Key, Key) bool
}

// generation demarcates Map clones. We use a heap-allocated reference
// instead of an integer to avoid integer overflows. Struct must have a field
// on it since two distinct zero-size variables may have the same address in
// memory.
type generation struct{ _ bool }

type Hasher interface {
	comparable
	Hash() uint64
}

// New returns a new empty Map.
func New[Key Hasher, Value any]() *Map[Key, Value] {
	return NewWithFuncs[Key, Value](func(k1, k2 Key) bool {
		return k1 == k2
	}, Key.Hash)
}

// NewWithFuncs is like New except that it uses explicit functions for comparison
// and hashing instead of relying on comparison and hashing on the value itself.
func NewWithFuncs[Key, Value any](
	eqFunc func(k1, k2 Key) bool,
	hashFunc func(Key) uint64,
) *Map[Key, Value] {
	if eqFunc == nil {
		var k Key
		switch (interface{}(k)).(type) {
		case string:
			eqFunc = interface{}(func(k1, k2 string) bool {
				return k1 == k2
			}).(func(Key, Key) bool)
		case []byte:
			eqFunc = interface{}(bytes.Equal).(func(Key, Key) bool)
		default:
			panic(fmt.Errorf("no equality type known for %T", k))
		}
	}
	if hashFunc == nil {
		var k Key
		switch (interface{}(k)).(type) {
		case string:
			hashFunc = interface{}(StringHash).(func(Key) uint64)
		case []byte:
			hashFunc = interface{}(BytesHash).(func(Key) uint64)
		default:
			panic(fmt.Errorf("no hash type known for %T", k))
		}
	}
	root := &iNode[Key, Value]{
		main: &mainNode[Key, Value]{
			cNode: &cNode[Key, Value]{},
		},
	}
	return newMap[Key, Value](root, eqFunc, hashFunc, false)
}

func newMap[Key, Value any](
	root *iNode[Key, Value],
	eqFunc func(Key, Key) bool,
	hashFunc func(Key) uint64,
	readOnly bool,
) *Map[Key, Value] {
	return &Map[Key, Value]{
		root:     root,
		eqFunc:   eqFunc,
		hashFunc: hashFunc,
		readOnly: readOnly,
	}
}

// Set sets the value for the given key, replacing the existing value if
// the key already exists.
func (c *Map[Key, Value]) Set(key Key, value Value) {
	c.assertReadWrite()
	c.insert(&mapEntry[Key, Value]{
		key:   key,
		value: value,
		hash:  uint32(c.hashFunc(key)),
	})
}

// Get returns the value for the associated key and
// reports whether the key exists in the trie.
func (c *Map[Key, Value]) Get(key Key) (Value, bool) {
	return c.lookup(&mapEntry[Key, Value]{
		key:  key,
		hash: uint32(c.hashFunc(key)),
	})
}

// Delete deletes the value for the associated key, returning
// the deleted value and returning true if an entry was removed.
func (c *Map[Key, Value]) Delete(key Key) (Value, bool) {
	c.assertReadWrite()
	return c.remove(&mapEntry[Key, Value]{
		key:  key,
		hash: uint32(c.hashFunc(key)),
	})
}

// Clone returns a stable, point-in-time clone of the Map. If the Map
// is read-only, the returned Map will also be read-only.
func (c *Map[Key, Value]) Clone() *Map[Key, Value] {
	return c.clone(c.readOnly)
}

// RClone returns a stable, point-in-time clone of the Map which
// is read-only. Write operations on a read-only clone will panic.
func (c *Map[Key, Value]) RClone() *Map[Key, Value] {
	return c.clone(true)
}

// clone wraps up the CAS logic to make a clone or a read-only clone.
func (c *Map[Key, Value]) clone(readOnly bool) *Map[Key, Value] {
	if readOnly && c.readOnly {
		return c
	}
	for {
		root := c.readRoot()
		main := gcasRead(root, c)
		if c.rdcssRoot(root, main, root.copyToGen(&generation{}, c)) {
			if readOnly {
				// For a read-only clone, we can share the old generation root.
				return newMap(root, c.eqFunc, c.hashFunc, readOnly)
			}
			// For a read-write clone, we need to take a copy of the root n the new generation.
			return newMap(c.readRoot().copyToGen(&generation{}, c), c.eqFunc, c.hashFunc, readOnly)
		}
	}
}

// Clear removes all keys from the Map.
func (c *Map[Key, Value]) Clear() {
	c.assertReadWrite()
	for {
		root := c.readRoot()
		gen := &generation{}
		newRoot := &iNode[Key, Value]{
			main: &mainNode[Key, Value]{cNode: &cNode[Key, Value]{gen: gen}},
			gen:  gen,
		}
		if c.rdcssRoot(root, gcasRead(root, c), newRoot) {
			return
		}
	}
}

// Len returns the number of keys in the Map.
// This operation is O(n).
func (c *Map[Key, Value]) Len() int {
	// TODO: The size operation can be optimized further by caching the size
	// information in main nodes of a read-only Map – this reduces the
	// amortized complexity of the size operation to O(1) because the size
	// computation is amortized across the update operations that occurred
	// since the last clone.
	size := 0
	for iter := c.Iterator(); iter.Next(); {
		size++
	}
	return size
}

// Iterator returns an iterator over the entries of the Map.
func (c *Map[Key, Value]) Iterator() *Iter[Key, Value] {
	iter := &Iter[Key, Value]{
		c: c,
	}
	iter.push((*Iter[Key, Value]).mainIter).iNode = c.RClone().readRoot()
	return iter
}

// Iter is an iterator that iterates through entries in the map.
type Iter[Key, Value any] struct {
	c *Map[Key, Value]
	// stack simulates the recursion stack that we'd have
	// if we were doing a conventional recursive iteration
	// through the data structure.
	stack []iterFrame[Key, Value]
	curr  *mapEntry[Key, Value]
}

type iterFrame[Key, Value any] struct {
	iter  func(*Iter[Key, Value], *iterFrame[Key, Value]) bool
	iNode *iNode[Key, Value]
	slice []branch
	lNode *lNode[Key, Value]
}

// TODO We'd like to define this as a type, but https://github.com/golang/go/issues/40060
// type iterFunc[Key, Value any] func(*Iter[Key, Value], *iterFrame[Key, Value]) bool

func (i *Iter[Key, Value]) Next() bool {
	i.curr = nil
	for i.curr == nil && len(i.stack) > 0 {
		if f := &i.stack[len(i.stack)-1]; !f.iter(i, f) {
			i.pop()
		}
	}
	return i.curr != nil
}

func (i *Iter[Key, Value]) Value() Value {
	if i.curr == nil {
		return z[Value]()
	}
	return i.curr.value
}

func (i *Iter[Key, Value]) Key() Key {
	if i.curr == nil {
		return z[Key]()
	}
	return i.curr.key
}

// mainIter iterates past a single iNode in the map.
func (i *Iter[Key, Value]) mainIter(f *iterFrame[Key, Value]) bool {
	if f.iNode == nil {
		return false
	}
	main := gcasRead(f.iNode, i.c)
	f.iNode = nil
	switch {
	case main.cNode != nil:
		i.push((*Iter[Key, Value]).sliceIter).slice = main.cNode.slice
		return true
	case main.lNode != nil:
		i.push((*Iter[Key, Value]).listIter).lNode = main.lNode
		return true
	case main.tNode != nil:
		i.curr = main.tNode.sNode.entry
		return true
	}
	panic("unreachable")
}

// sliceIter iterates through the entries in a cNode.
func (i *Iter[Key, Value]) sliceIter(f *iterFrame[Key, Value]) bool {
	a := f.slice
	if len(a) == 0 {
		return false
	}
	f.slice = a[1:]
	switch b := a[0].(type) {
	case *iNode[Key, Value]:
		i.push((*Iter[Key, Value]).mainIter).iNode = b
		return true
	case *sNode[Key, Value]:
		i.curr = b.entry
		return true
	}
	panic("unreachable")
}

// listIter iterates through the list of entries in an lNode.
func (i *Iter[Key, Value]) listIter(f *iterFrame[Key, Value]) bool {
	l := f.lNode
	if l == nil {
		return false
	}
	f.lNode = f.lNode.tail
	i.curr = l.head.entry
	return true
}

// pop pops a value off the iterator stack.
func (i *Iter[Key, Value]) pop() {
	i.stack = i.stack[0 : len(i.stack)-1]
}

// push pushes the given iteration function onto the iterator stack
// and returns the new frame.
// The caller is responsible for setting up the frame appropriately
// for the iteration function.
func (i *Iter[Key, Value]) push(f func(*Iter[Key, Value], *iterFrame[Key, Value]) bool) *iterFrame[Key, Value] {
	i.stack = append(i.stack, iterFrame[Key, Value]{})
	elem := &i.stack[len(i.stack)-1]
	elem.iter = f
	return elem
}

func (c *Map[Key, Value]) assertReadWrite() {
	if c.readOnly {
		panic("Cannot modify read-only clone")
	}
}

func (c *Map[Key, Value]) insert(entry *mapEntry[Key, Value]) {
	root := c.readRoot()
	if !c.iinsert(root, entry, 0, nil, root.gen) {
		c.insert(entry)
	}
}

func (c *Map[Key, Value]) lookup(entry *mapEntry[Key, Value]) (Value, bool) {
	root := c.readRoot()
	result, exists, ok := c.ilookup(root, entry, 0, nil, root.gen)
	for !ok {
		return c.lookup(entry)
	}
	return result, exists
}

func (c *Map[Key, Value]) remove(entry *mapEntry[Key, Value]) (Value, bool) {
	root := c.readRoot()
	result, exists, ok := c.iremove(root, entry, 0, nil, root.gen)
	for !ok {
		return c.remove(entry)
	}
	return result, exists
}

// iinsert attempts to insert the entry into the Map. If false is returned,
// the operation should be retried.
func (c *Map[Key, Value]) iinsert(i *iNode[Key, Value], entry *mapEntry[Key, Value], lev uint, parent *iNode[Key, Value], startGen *generation) bool {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		cn := main.cNode
		flag, pos := flagPos(entry.hash, lev, cn.bmp)
		if cn.bmp&flag == 0 {
			// If the relevant bit is not in the bitmap, then a copy of the
			// cNode with the new entry is created. The linearization point is
			// a successful CAS.
			rn := cn
			if cn.gen != i.gen {
				rn = cn.renewed(i.gen, c)
			}
			ncn := &mainNode[Key, Value]{
				cNode: rn.inserted(pos, flag, &sNode[Key, Value]{entry}, i.gen),
			}
			return gcas(i, main, ncn, c)
		}
		// If the relevant bit is present in the bitmap, then its corresponding
		// branch is read from the slice.
		branch := cn.slice[pos]
		switch branch := branch.(type) {
		case *iNode[Key, Value]:
			// If the branch is an I-node, then iinsert is called recursively.
			if startGen == branch.gen {
				return c.iinsert(branch, entry, lev+w, i, startGen)
			}
			if gcas(i, main, &mainNode[Key, Value]{cNode: cn.renewed(startGen, c)}, c) {
				return c.iinsert(i, entry, lev, parent, startGen)
			}
			return false
		case *sNode[Key, Value]:
			sn := branch
			if !c.eqFunc(sn.entry.key, entry.key) {
				// If the branch is an S-node and its key is not equal to the
				// key being inserted, then the Map has to be extended with
				// an additional level. The C-node is replaced with its updated
				// version, created using the updated function that adds a new
				// I-node at the respective position. The new Inode has its
				// main node pointing to a C-node with both keys. The
				// linearization point is a successful CAS.
				rn := cn
				if cn.gen != i.gen {
					rn = cn.renewed(i.gen, c)
				}
				nsn := &sNode[Key, Value]{entry}
				nin := &iNode[Key, Value]{main: newMainNode(sn, sn.entry.hash, nsn, nsn.entry.hash, lev+w, i.gen), gen: i.gen}
				ncn := &mainNode[Key, Value]{cNode: rn.updated(pos, nin, i.gen)}
				return gcas(i, main, ncn, c)
			}
			// If the key in the S-node is equal to the key being inserted,
			// then the C-node is replaced with its updated version with a new
			// S-node. The linearization point is a successful CAS.
			ncn := &mainNode[Key, Value]{cNode: cn.updated(pos, &sNode[Key, Value]{entry}, i.gen)}
			return gcas(i, main, ncn, c)
		default:
			panic("Map is in an invalid state")
		}
	case main.tNode != nil:
		clean(parent, lev-w, c)
		return false
	case main.lNode != nil:
		nln := &mainNode[Key, Value]{lNode: main.lNode.inserted(entry, c.eqFunc)}
		return gcas(i, main, nln, c)
	default:
		panic("Map is in an invalid state")
	}
}

// ilookup attempts to fetch the entry from the Map. The first two return
// values are the entry value and whether or not the entry was contained in the
// Map. The last bool indicates if the operation succeeded. False means it
// should be retried.
func (c *Map[Key, Value]) ilookup(i *iNode[Key, Value], entry *mapEntry[Key, Value], lev uint, parent *iNode[Key, Value], startGen *generation) (Value, bool, bool) {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		cn := main.cNode
		flag, pos := flagPos(entry.hash, lev, cn.bmp)
		if cn.bmp&flag == 0 {
			// If the bitmap does not contain the relevant bit, a key with the
			// required hashcode prefix is not present in the trie.
			return z[Value](), false, true
		}
		// Otherwise, the relevant branch at index pos is read from the slice.
		branch := cn.slice[pos]
		switch branch := branch.(type) {
		case *iNode[Key, Value]:
			// If the branch is an I-node, the ilookup procedure is called
			// recursively at the next level.
			in := branch
			if c.readOnly || startGen == in.gen {
				return c.ilookup(in, entry, lev+w, i, startGen)
			}
			if gcas(i, main, &mainNode[Key, Value]{cNode: cn.renewed(startGen, c)}, c) {
				return c.ilookup(i, entry, lev, parent, startGen)
			}
			return z[Value](), false, false
		case *sNode[Key, Value]:
			// If the branch is an S-node, then the key within the S-node is
			// compared with the key being searched – these two keys have the
			// same hashcode prefixes, but they need not be equal. If they are
			// equal, the corresponding value from the S-node is
			// returned and a NOTFOUND value otherwise.
			sn := branch
			if c.eqFunc(sn.entry.key, entry.key) {
				return sn.entry.value, true, true
			}
			return z[Value](), false, true
		default:
			panic("Map is in an invalid state")
		}
	case main.tNode != nil:
		return cleanReadOnly(main.tNode, lev, parent, c, entry)
	case main.lNode != nil:
		// Hash collisions are handled using L-nodes, which are essentially
		// persistent linked lists.
		val, ok := main.lNode.lookup(entry, c.eqFunc)
		return val, ok, true
	default:
		panic("Map is in an invalid state")
	}
}

// iremove attempts to remove the entry from the Map. The first two return
// values are the entry value and whether or not the entry was contained in the
// Map. The last bool indicates if the operation succeeded. False means it
// should be retried.
func (c *Map[Key, Value]) iremove(i *iNode[Key, Value], entry *mapEntry[Key, Value], lev uint, parent *iNode[Key, Value], startGen *generation) (Value, bool, bool) {
	// Linearization point.
	main := gcasRead(i, c)
	switch {
	case main.cNode != nil:
		cn := main.cNode
		flag, pos := flagPos(entry.hash, lev, cn.bmp)
		if cn.bmp&flag == 0 {
			// If the bitmap does not contain the relevant bit, a key with the
			// required hashcode prefix is not present in the trie.
			return z[Value](), false, true
		}
		// Otherwise, the relevant branch at index pos is read from the slice.
		branch := cn.slice[pos]
		switch branch := branch.(type) {
		case *iNode[Key, Value]:
			// If the branch is an I-node, the iremove procedure is called
			// recursively at the next level.
			in := branch
			if startGen == in.gen {
				return c.iremove(in, entry, lev+w, i, startGen)
			}
			if gcas(i, main, &mainNode[Key, Value]{cNode: cn.renewed(startGen, c)}, c) {
				return c.iremove(i, entry, lev, parent, startGen)
			}
			return z[Value](), false, false
		case *sNode[Key, Value]:
			// If the branch is an S-node, its key is compared against the key
			// being removed.
			sn := branch
			if !c.eqFunc(sn.entry.key, entry.key) {
				// If the keys are not equal, the NOTFOUND value is returned.
				return z[Value](), false, true
			}
			//  If the keys are equal, a copy of the current node without the
			//  S-node is created. The contraction of the copy is then created
			//  using the toContracted procedure. A successful CAS will
			//  substitute the old C-node with the copied C-node, thus removing
			//  the S-node with the given key from the trie – this is the
			//  linearization point
			ncn := cn.removed(pos, flag, i.gen)
			cntr := toContracted(ncn, lev)
			if gcas(i, main, cntr, c) {
				if parent != nil {
					main = gcasRead(i, c)
					if main.tNode != nil {
						cleanParent(parent, i, entry.hash, lev-w, c, startGen)
					}
				}
				return sn.entry.value, true, true
			}
			return z[Value](), false, false
		default:
			panic("Map is in an invalid state")
		}
	case main.tNode != nil:
		clean(parent, lev-w, c)
		return z[Value](), false, false
	case main.lNode != nil:
		nln := &mainNode[Key, Value]{
			lNode: main.lNode.removed(entry, c.eqFunc),
		}
		if nln.lNode != nil && nln.lNode.tail == nil {
			// Exactly one entry.
			nln = entomb(nln.lNode.head)
		}
		if gcas(i, main, nln, c) {
			val, ok := main.lNode.lookup(entry, c.eqFunc)
			return val, ok, true
		}
		return z[Value](), false, true
	default:
		panic("Map is in an invalid state")
	}
}

// iNode is an indirection node. I-nodes remain present in the Map even as
// nodes above and below change. Thread-safety is achieved in part by
// performing CAS operations on the I-node instead of the internal node slice.
type iNode[Key, Value any] struct {
	main *mainNode[Key, Value]
	gen  *generation

	// rdcss is set during an RDCSS operation. The I-node is actually a wrapper
	// around the descriptor in this case so that a single type is used during
	// CAS operations on the root.
	rdcss *rdcssDescriptor[Key, Value]
}

// copyToGen returns a copy of this I-node copied to the given generation.
func (i *iNode[Key, Value]) copyToGen(gen *generation, ctrie *Map[Key, Value]) *iNode[Key, Value] {
	nin := &iNode[Key, Value]{gen: gen}
	main := gcasRead(i, ctrie)
	gatomic.StorePointer(&nin.main, main)
	return nin
}

// mainNode is either a cNode, tNode, lNode, or failed node which makes up an
// I-node.
type mainNode[Key, Value any] struct {
	cNode  *cNode[Key, Value]
	tNode  *tNode[Key, Value]
	lNode  *lNode[Key, Value]
	failed *mainNode[Key, Value]

	// prev is set as a failed main node when we attempt to CAS and the
	// I-node's generation does not match the root generation. This signals
	// that the GCAS failed and the I-node's main node must be set back to the
	// previous value.
	prev *mainNode[Key, Value]
}

// cNode is an internal main node containing a bitmap and the slice with
// references to branch nodes. A branch node is either another I-node or a
// singleton S-node.
type cNode[Key, Value any] struct {
	bmp   uint32
	slice []branch
	gen   *generation
}

// newMainNode is a recursive constructor which creates a new mainNode. This
// mainNode will consist of cNodes as long as the hashcode chunks of the two
// keys are equal at the given level. If the level exceeds 2^w, an lNode is
// created.
func newMainNode[Key, Value any](x *sNode[Key, Value], xhc uint32, y *sNode[Key, Value], yhc uint32, lev uint, gen *generation) *mainNode[Key, Value] {
	if lev >= exp2 {
		return &mainNode[Key, Value]{
			lNode: &lNode[Key, Value]{
				head: y,
				tail: &lNode[Key, Value]{
					head: x,
				},
			},
		}
	}
	xidx := (xhc >> lev) & 0x1f
	yidx := (yhc >> lev) & 0x1f
	bmp := uint32((1 << xidx) | (1 << yidx))

	switch {
	case xidx == yidx:
		// Recurse when indexes are equal.
		main := newMainNode(x, xhc, y, yhc, lev+w, gen)
		iNode := &iNode[Key, Value]{main: main, gen: gen}
		return &mainNode[Key, Value]{cNode: &cNode[Key, Value]{bmp, []branch{iNode}, gen}}
	case xidx < yidx:
		return &mainNode[Key, Value]{cNode: &cNode[Key, Value]{bmp, []branch{x, y}, gen}}
	default:
		return &mainNode[Key, Value]{cNode: &cNode[Key, Value]{bmp, []branch{y, x}, gen}}
	}
}

// inserted returns a copy of this cNode with the new entry at the given position.
func (c *cNode[Key, Value]) inserted(pos int, flag uint32, br branch, gen *generation) *cNode[Key, Value] {
	slice := make([]branch, len(c.slice)+1)
	copy(slice, c.slice[:pos])
	slice[pos] = br
	copy(slice[pos+1:], c.slice[pos:])
	return &cNode[Key, Value]{
		bmp:   c.bmp | flag,
		slice: slice,
		gen:   gen,
	}
}

// updated returns a copy of this cNode with the entry at the given index updated.
func (c *cNode[Key, Value]) updated(pos int, br branch, gen *generation) *cNode[Key, Value] {
	slice := make([]branch, len(c.slice))
	copy(slice, c.slice)
	slice[pos] = br
	return &cNode[Key, Value]{
		bmp:   c.bmp,
		slice: slice,
		gen:   gen,
	}
}

// removed returns a copy of this cNode with the entry at the given index
// removed.
func (c *cNode[Key, Value]) removed(pos int, flag uint32, gen *generation) *cNode[Key, Value] {
	slice := make([]branch, len(c.slice)-1)
	copy(slice, c.slice[0:pos])
	copy(slice[pos:], c.slice[pos+1:])
	return &cNode[Key, Value]{
		bmp:   c.bmp ^ flag,
		slice: slice,
		gen:   gen,
	}
}

// renewed returns a copy of this cNode with the I-nodes below it copied to the
// given generation.
func (c *cNode[Key, Value]) renewed(gen *generation, ctrie *Map[Key, Value]) *cNode[Key, Value] {
	slice := make([]branch, len(c.slice))
	for i, br := range c.slice {
		switch t := br.(type) {
		case *iNode[Key, Value]:
			slice[i] = t.copyToGen(gen, ctrie)
		default:
			slice[i] = br
		}
	}
	return &cNode[Key, Value]{
		bmp:   c.bmp,
		slice: slice,
		gen:   gen,
	}
}

// tNode is tomb node which is a special node used to ensure proper ordering
// during removals.
type tNode[Key, Value any] struct {
	sNode *sNode[Key, Value]
}

// untombed returns the S-node contained by the T-node.
func (t *tNode[Key, Value]) untombed() *sNode[Key, Value] {
	return &sNode[Key, Value]{&mapEntry[Key, Value]{
		key:   t.sNode.entry.key,
		value: t.sNode.entry.value,
		hash:  t.sNode.entry.hash,
	}}
}

// lNode is a list node which is a leaf node used to handle hashcode
// collisions by keeping such keys in a persistent list.
type lNode[Key, Value any] struct {
	head *sNode[Key, Value]
	tail *lNode[Key, Value]
}

// lookup returns the value at the given entry in the L-node or returns false
// if it's not contained.
func (l *lNode[Key, Value]) lookup(e *mapEntry[Key, Value], eq func(Key, Key) bool) (Value, bool) {
	for ; l != nil; l = l.tail {
		if eq(e.key, l.head.entry.key) {
			return l.head.entry.value, true
		}
	}
	return z[Value](), false
}

// inserted creates a new L-node with the added entry.
func (l *lNode[Key, Value]) inserted(entry *mapEntry[Key, Value], eq func(Key, Key) bool) *lNode[Key, Value] {
	return &lNode[Key, Value]{
		head: &sNode[Key, Value]{entry},
		tail: l.removed(entry, eq),
	}
}

// removed creates a new L-node with the entry removed.
func (l *lNode[Key, Value]) removed(e *mapEntry[Key, Value], eq func(Key, Key) bool) *lNode[Key, Value] {
	for l1 := l; l1 != nil; l1 = l1.tail {
		if eq(e.key, l1.head.entry.key) {
			return l.remove(l1)
		}
	}
	return l
}

func (l *lNode[Key, Value]) remove(l1 *lNode[Key, Value]) *lNode[Key, Value] {
	if l == l1 {
		return l.tail
	}
	return &lNode[Key, Value]{
		head: l.head,
		tail: l.tail.remove(l1),
	}
}

// branch is either *iNode or *sNode.
type branch interface{}

// mapEntry contains a Map key-value pair.
type mapEntry[Key, Value any] struct {
	key   Key
	value Value
	hash  uint32
}

// sNode is a singleton node which contains a single key and value.
type sNode[Key, Value any] struct {
	entry *mapEntry[Key, Value]
}

// toContracted ensures that every I-node except the root points to a C-node
// with at least one branch. If a given C-Node has only a single S-node below
// it and is not at the root level, a T-node which wraps the S-node is
// returned.
func toContracted[Key, Value any](cn *cNode[Key, Value], lev uint) *mainNode[Key, Value] {
	if lev > 0 && len(cn.slice) == 1 {
		switch branch := cn.slice[0].(type) {
		case *sNode[Key, Value]:
			return entomb(branch)
		default:
			return &mainNode[Key, Value]{cNode: cn}
		}
	}
	return &mainNode[Key, Value]{cNode: cn}
}

// toCompressed compacts the C-node as a performance optimization.
func toCompressed[Key, Value any](cn *cNode[Key, Value], lev uint) *mainNode[Key, Value] {
	tmpSlice := make([]branch, len(cn.slice))
	for i, sub := range cn.slice {
		switch sub := sub.(type) {
		case *iNode[Key, Value]:
			inode := sub
			main := gatomic.LoadPointer(&inode.main)
			tmpSlice[i] = resurrect(inode, main)
		case *sNode[Key, Value]:
			tmpSlice[i] = sub
		default:
			panic("Map is in an invalid state")
		}
	}

	return toContracted(&cNode[Key, Value]{
		bmp:   cn.bmp,
		slice: tmpSlice,
	}, lev)
}

func entomb[Key, Value any](m *sNode[Key, Value]) *mainNode[Key, Value] {
	return &mainNode[Key, Value]{tNode: &tNode[Key, Value]{m}}
}

func resurrect[Key, Value any](iNode *iNode[Key, Value], main *mainNode[Key, Value]) branch {
	if main.tNode != nil {
		return main.tNode.untombed()
	}
	return iNode
}

func clean[Key, Value any](i *iNode[Key, Value], lev uint, ctrie *Map[Key, Value]) bool {
	main := gcasRead(i, ctrie)
	if main.cNode != nil {
		return gcas(i, main, toCompressed(main.cNode, lev), ctrie)
	}
	return true
}

func cleanReadOnly[Key, Value any](tn *tNode[Key, Value], lev uint, p *iNode[Key, Value], ctrie *Map[Key, Value], entry *mapEntry[Key, Value]) (val Value, exists bool, ok bool) {
	if !ctrie.readOnly {
		clean(p, lev-5, ctrie)
		return z[Value](), false, false
	}
	if tn.sNode.entry.hash == entry.hash && ctrie.eqFunc(tn.sNode.entry.key, entry.key) {
		return tn.sNode.entry.value, true, true
	}
	return z[Value](), false, true
}

func cleanParent[Key, Value any](p, i *iNode[Key, Value], hc uint32, lev uint, ctrie *Map[Key, Value], startGen *generation) {
	main := gatomic.LoadPointer(&i.main)
	pMain := gatomic.LoadPointer(&p.main)
	if pMain.cNode == nil {
		return
	}
	flag, pos := flagPos(hc, lev, pMain.cNode.bmp)
	if pMain.cNode.bmp&flag == 0 {
		return
	}
	sub := pMain.cNode.slice[pos]
	if sub != i || main.tNode == nil {
		return
	}
	ncn := pMain.cNode.updated(pos, resurrect(i, main), i.gen)
	if gcas(p, pMain, toContracted(ncn, lev), ctrie) || ctrie.readRoot().gen != startGen {
		return
	}
	cleanParent(p, i, hc, lev, ctrie, startGen)
}

func flagPos(hashcode uint32, lev uint, bmp uint32) (uint32, int) {
	idx := (hashcode >> lev) & 0x1f
	flag := uint32(1) << idx
	pos := bits.OnesCount32(bmp & (flag - 1))
	return flag, pos
}

// gcas is a generation-compare-and-swap which has semantics similar to RDCSS,
// but it does not create the intermediate object except in the case of
// failures that occur due to the clone being taken. This ensures that the
// write occurs only if the Map root generation has remained the same in
// addition to the I-node having the expected value.
func gcas[Key, Value any](in *iNode[Key, Value], old, n *mainNode[Key, Value], ct *Map[Key, Value]) bool {
	gatomic.StorePointer(&n.prev, old)
	if gatomic.CompareAndSwapPointer(&in.main, old, n) {
		gcasComplete(in, n, ct)
		return gatomic.LoadPointer(&n.prev) == nil
	}
	return false
}

// gcasRead performs a GCAS-linearizable read of the I-node's main node.
func gcasRead[Key, Value any](in *iNode[Key, Value], ctrie *Map[Key, Value]) *mainNode[Key, Value] {
	m := gatomic.LoadPointer(&in.main)
	if gatomic.LoadPointer(&m.prev) == nil {
		return m
	}
	return gcasComplete(in, m, ctrie)
}

// gcasComplete commits the GCAS operation.
func gcasComplete[Key, Value any](i *iNode[Key, Value], m *mainNode[Key, Value], ctrie *Map[Key, Value]) *mainNode[Key, Value] {
	for {
		if m == nil {
			return nil
		}
		prev := gatomic.LoadPointer(&m.prev)
		root := ctrie.rdcssReadRoot(true)
		if prev == nil {
			return m
		}

		if prev.failed != nil {
			// Signals GCAS failure. Swap old value back into I-node.
			fn := prev.failed
			if gatomic.CompareAndSwapPointer(&i.main, m, fn) {
				return fn
			}
			m = gatomic.LoadPointer(&i.main)
			continue
		}

		if root.gen == i.gen && !ctrie.readOnly {
			// Commit GCAS.
			if gatomic.CompareAndSwapPointer(&m.prev, prev, nil) {
				return m
			}
			continue
		}

		// Generations did not match. Store failed node on prev to signal
		// I-node's main node must be set back to the previous value.
		gatomic.CompareAndSwapPointer(&m.prev, prev, &mainNode[Key, Value]{failed: prev})
		m = gatomic.LoadPointer(&i.main)
		return gcasComplete(i, m, ctrie)
	}
}

// rdcssDescriptor is an intermediate struct which communicates the intent to
// replace the value in an I-node and check that the root's generation has not
// changed before committing to the new value.
type rdcssDescriptor[Key, Value any] struct {
	old       *iNode[Key, Value]
	expected  *mainNode[Key, Value]
	nv        *iNode[Key, Value]
	committed int32
}

// readRoot performs a linearizable read of the Map root. This operation is
// prioritized so that if another thread performs a GCAS on the root, a
// deadlock does not occur.
func (c *Map[Key, Value]) readRoot() *iNode[Key, Value] {
	return c.rdcssReadRoot(false)
}

// rdcssReadRoot performs a RDCSS-linearizable read of the Map root with the
// given priority.
func (c *Map[Key, Value]) rdcssReadRoot(abort bool) *iNode[Key, Value] {
	r := gatomic.LoadPointer(&c.root)
	if r.rdcss != nil {
		return c.rdcssComplete(abort)
	}
	return r
}

// rdcssRoot performs a RDCSS on the Map root. This is used to create a
// clone of the Map by copying the root I-node and setting it to a new
// generation.
func (c *Map[Key, Value]) rdcssRoot(old *iNode[Key, Value], expected *mainNode[Key, Value], nv *iNode[Key, Value]) bool {
	desc := &iNode[Key, Value]{
		rdcss: &rdcssDescriptor[Key, Value]{
			old:      old,
			expected: expected,
			nv:       nv,
		},
	}
	if c.casRoot(old, desc) {
		c.rdcssComplete(false)
		return gatomic.LoadInt32(&desc.rdcss.committed) == 1
	}
	return false
}

// rdcssComplete commits the RDCSS operation.
func (c *Map[Key, Value]) rdcssComplete(abort bool) *iNode[Key, Value] {
	for {
		r := gatomic.LoadPointer(&c.root)
		if r.rdcss == nil {
			return r
		}
		desc := r.rdcss
		ov := desc.old
		exp := desc.expected
		nv := desc.nv
		if abort {
			if c.casRoot(r, ov) {
				return ov
			}
			continue
		}
		oldeMain := gcasRead(ov, c)
		if oldeMain == exp {
			// Commit the RDCSS.
			if c.casRoot(r, nv) {
				gatomic.StoreInt32(&desc.committed, 1)
				return nv
			}
			continue
		}
		if c.casRoot(r, ov) {
			return ov
		}
	}
}

// casRoot performs a CAS on the Map root.
func (c *Map[Key, Value]) casRoot(ov, nv *iNode[Key, Value]) bool {
	c.assertReadWrite()
	return gatomic.CompareAndSwapPointer(&c.root, ov, nv)
}

// z returns the zero value of V.
func z[V any]() V {
	var v V
	return v
}
