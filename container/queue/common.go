package queue

import (
	"sync/atomic"
	"unsafe"
)

const CacheLineSize = 128
const (
	MaskLittle32 = uint64((1 << 32) - 1)
	MaskBig32    = ^MaskLittle32
)

type chunk64[T any] struct {
	next    *chunk64[T]
	_       [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	headIdx int64
	_       [CacheLineSize - 8]byte
	tailIdx int64
	slot    []element[T]
}

type element[T any] struct {
	//_    [60]byte
	elt  T
	flag int64
	//_    [60]byte
}

//type element_pad[T any] struct {
//	element[T]
//	pad [cacheLineSize - unsafe.Sizeof(element[T]{})%cacheLineSize]byte
//}

func newChunk64[T any](size int64) *chunk64[T] {
	return &chunk64[T]{slot: make([]element[T], size)}
}

func atomicLoadChunk64[T any](addr **chunk64[T]) *chunk64[T] {
	return (*chunk64[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(addr))))
}

func atomicStoreChunk64[T any](addr **chunk64[T], ptr *chunk64[T]) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(ptr))
}

func atomicSwapChunk64[T any](addr **chunk64[T], new *chunk64[T]) (old *chunk64[T]) {
	return (*chunk64[T])(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(new)))
}

func atomicSwapElementPtr[T any](addr **element[T], new *element[T]) (old *element[T]) {
	return (*element[T])(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(new)))
}

func atomicCASElementPtr[T any](addr **element[T], old, new *element[T]) bool {
	return atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(old), unsafe.Pointer(new))
}

func circularLessThanU64(a, b uint64) bool {
	return (a - b) < uint64(1)<<63
}
