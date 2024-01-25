package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	util_math "github.com/panjf2000/gnet/v2/util/math"
)

type spmcChunk[T any] struct {
	ref     int64
	next    *spmcChunk[T]
	_       [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	headIdx int64
	_       [CacheLineSize - 8]byte
	tailIdx int64
	slot    []T
}

func newChunk_spmc[T any](size int64) *spmcChunk[T] {
	return &spmcChunk[T]{
		ref:  size,
		slot: make([]T, size),
	}
}

func atomicLoadChunk_spmc[T any](addr **spmcChunk[T]) *spmcChunk[T] {
	return (*spmcChunk[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(addr))))
}

func atomicStoreChunk_spmc[T any](addr **spmcChunk[T], ptr *spmcChunk[T]) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(ptr))
}

func atomicSwapChunk_spmc[T any](addr **spmcChunk[T], new *spmcChunk[T]) (old *spmcChunk[T]) {
	return (*spmcChunk[T])(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(new)))
}

type spmcChain[T any] struct {
	num       atomic.Int64
	_         [CacheLineSize - unsafe.Sizeof(atomic.Int64{})]byte
	headChunk *spmcChunk[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	tailChunk *spmcChunk[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	memCache  *spmcChunk[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	ch        chan struct{}
	chunkSize int64
}

func newChain_spmc[T any](size int64) *spmcChain[T] {
	if size < 0 {
		panic(fmt.Errorf("invalid size %d", size))
	}

	chunk := newChunk_spmc[T](size)
	return &spmcChain[T]{
		ch:        make(chan struct{}, 1),
		headChunk: chunk,
		tailChunk: chunk,
		chunkSize: size,
	}
}

func (inst *spmcChain[T]) pushTail(v T) {
	tailChunk := inst.tailChunk
	tailChunk.slot[tailChunk.tailIdx] = v

	tailChunk.tailIdx++
	if tailChunk.tailIdx == inst.chunkSize {
		chunk := atomicSwapChunk_spmc[T](&inst.memCache, nil)
		if chunk == nil {
			chunk = newChunk_spmc[T](inst.chunkSize)

		} else {
			chunk.tailIdx = 0
		}

		tailChunk.next = chunk
		// 让consumer可以读到最新的headChunk.next
		atomicStoreChunk_spmc[T](&inst.tailChunk, chunk)
	}

	if inst.num.Add(1) < 1 {
		inst.ch <- struct{}{}
	}
}

func (inst *spmcChain[T]) popHead() T {
	if inst.num.Add(-1) < 0 {
		<-inst.ch
	}

RETRY:
	headChunk := atomicLoadChunk_spmc[T](&inst.headChunk)
	// 此时的headChunk很有可能已被其它consumer挂到了memCache
	hi := atomic.AddInt64(&headChunk.headIdx, 1) // incr-1
	if hi > inst.chunkSize {
		goto RETRY
	}

	v := headChunk.slot[hi-1]
	if atomic.AddInt64(&headChunk.ref, -1) == 0 {
		chunk := atomicLoadChunk_spmc(&headChunk.next)
		chunk.headIdx = 0
		chunk.ref = inst.chunkSize

		// 此时不重置headIdx和ref，避免和incr-1冲突
		headChunk.next = nil
		atomicStoreChunk_spmc[T](&inst.headChunk, chunk)
		atomicSwapChunk_spmc[T](&inst.memCache, headChunk)
	}

	return v
}

type spmcRing[T any] struct {
	num     atomic.Int64
	_       [CacheLineSize - unsafe.Sizeof(atomic.Int64{})]byte
	headIdx int64
	_       [CacheLineSize - 8]byte
	tailIdx int64
	_       [CacheLineSize - 8]byte
	slot    []element[T]
	cap     int64
	mod     int64
	ch      chan struct{}
}

func newRing_spmc[T any](size int64) *spmcRing[T] {
	if size < 0 {
		panic(fmt.Errorf("size MUST greater than 0, got %d", size))
	}

	size = int64(util_math.CeilToPowerOfTwo(uint64(size)))
	return &spmcRing[T]{
		slot: make([]element[T], size),
		cap:  size,
		mod:  size - 1,
		ch:   make(chan struct{}, 1),
	}
}

func (inst *spmcRing[T]) pushTail(v T) {
	pSlot := &inst.slot[inst.tailIdx&inst.mod]
	for atomic.LoadInt64(&pSlot.flag) > 0 {
		runtime.Gosched()
	}

	pSlot.elt = v
	pSlot.flag = 1
	inst.tailIdx++

	if inst.num.Add(1) < 1 {
		inst.ch <- struct{}{}
	}
}

func (inst *spmcRing[T]) popHead() T {
	if inst.num.Add(-1) < 0 {
		<-inst.ch
	}

	hi := atomic.AddInt64(&inst.headIdx, 1) - 1
	pSlot := &inst.slot[hi&inst.mod]

	v := pSlot.elt
	atomic.StoreInt64(&pSlot.flag, 0)

	return v
}
