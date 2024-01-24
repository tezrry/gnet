package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	util_math "github.com/panjf2000/gnet/v2/util/math"
)

type spmcChain[T any] struct {
	num       atomic.Int64
	_         [CacheLineSize - unsafe.Sizeof(atomic.Int64{})]byte
	headChunk *chunk64[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	tailChunk *chunk64[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	memCache  *chunk64[T]
	_         [CacheLineSize - unsafe.Sizeof((*int)(nil))]byte
	ch        chan struct{}
	chunkSize int64
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

func newChain_spmc[T any](size int64) *spmcChain[T] {
	chunk := newChunk64[T](size)
	return &spmcChain[T]{
		ch:        make(chan struct{}, 1),
		headChunk: chunk,
		tailChunk: chunk,
		chunkSize: size,
	}
}

func (inst *spmcChain[T]) pushTail(v T) {
	tailChunk := inst.tailChunk
	pSlot := &tailChunk.slot[tailChunk.tailIdx]
	pSlot.elt = v
	pSlot.flag = 1

	tailChunk.tailIdx++
	if tailChunk.tailIdx == inst.chunkSize {
		chunk := atomicSwapChunk64[T](&inst.memCache, nil)
		if chunk == nil {
			chunk = newChunk64[T](inst.chunkSize)

		} else {
			chunk.tailIdx = 0
			//atomic.StoreInt64(&chunk.headIdx, 0)
			//chunk.headIdx = 0
		}

		tailChunk.next = chunk
		// 让consumer可以读到最新的headChunk.next
		atomicStoreChunk64[T](&inst.tailChunk, chunk)
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
	headChunk := atomicLoadChunk64[T](&inst.headChunk)
	hi := atomic.AddInt64(&headChunk.headIdx, 1)
	if hi > inst.chunkSize {
		headChunk = atomicLoadChunk64[T](&inst.headChunk.next)
		goto RETRY
	}

	pSlot := &headChunk.slot[hi-1]
	//if atomic.LoadInt64(&pSlot.flag) == 0 {
	//	goto RETRY
	//}

	v := pSlot.elt
	pSlot.flag = 0
	//atomic.StoreInt64(&pSlot.flag, 0)

	//if hi == inst.chunkSize {
	//	chunk := atomicLoadChunk64(&headChunk.next)
	//	headChunk.next = nil
	//	atomic.StoreInt64(&headChunk.headIdx, 0)
	//	atomicStoreChunk64[T](&inst.headChunk, chunk)
	//	atomicSwapChunk64[T](&inst.memCache, headChunk)
	//}

	return v
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
