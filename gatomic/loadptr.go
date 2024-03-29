package gatomic

import (
	"sync/atomic"
	"unsafe"
)

func LoadPointer[T any](addr **T) *T {
	return (*T)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(addr))))
}

func StorePointer[T any](addr **T, val *T) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(val))
}

func CompareAndSwapPointer[T any](addr **T, old, new *T) (swapped bool) {
	return atomic.CompareAndSwapPointer(
		(*unsafe.Pointer)(unsafe.Pointer(addr)),
		unsafe.Pointer(old),
		unsafe.Pointer(new),
	)
}

func LoadInt32(x *int32) int32 {
	return atomic.LoadInt32(x)
}

func StoreInt32(x *int32, v int32) {
	atomic.StoreInt32(x, v)
}
