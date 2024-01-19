package queue

import (
	"runtime"
	"sync/atomic"
	"unsafe"
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
	for atomic.LoadInt64(&pSlot.flag) > 0 {
		runtime.Gosched()
	}

	pSlot.elt = v
	pSlot.flag = 1

	tailChunk.tailIdx++
	if tailChunk.tailIdx == inst.chunkSize {
		chunk := atomicSwapChunk64[T](&inst.memCache, nil)
		if chunk == nil {
			chunk = newChunk64[T](inst.chunkSize)

		} else {
			chunk.tailIdx = 0
		}

		//chunk.prev = tailChunk
		tailChunk.next = chunk
		inst.tailChunk = chunk
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
		runtime.Gosched()
		goto RETRY
	}

	pSlot := &headChunk.slot[hi-1]
	v := pSlot.elt
	atomic.StoreInt64(&pSlot.flag, 0)

	if hi == inst.chunkSize {
		chunk := headChunk.next
		//chunk.prev = nil
		headChunk.next = nil
		atomic.StoreInt64(&headChunk.headIdx, 0)
		atomicStoreChunk64[T](&inst.headChunk, chunk)
		atomicSwapChunk64[T](&inst.memCache, headChunk)
	}

	return v
}
