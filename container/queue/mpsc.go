package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/panjf2000/gnet/v2/link"
	util_math "github.com/panjf2000/gnet/v2/util/math"
)

const consumer_spin_cnt = uint32(30)

type mpscChain[T any] struct {
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

type mpscRing[T any] struct {
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

func newChain_mpsc[T any](size int64) *mpscChain[T] {
	chunk := newChunk64[T](size)
	return &mpscChain[T]{
		ch:        make(chan struct{}, 1),
		headChunk: chunk,
		tailChunk: chunk,
		chunkSize: size,
	}
}

func newRing_mpsc[T any](size int64) *mpscRing[T] {
	if size < 0 {
		panic(fmt.Errorf("size MUST greater than 0, got %d", size))
	}

	size = int64(util_math.CeilToPowerOfTwo(uint64(size)))
	return &mpscRing[T]{
		slot: make([]element[T], size),
		cap:  size,
		mod:  size - 1,
		ch:   make(chan struct{}, 1),
	}
}

func (inst *mpscChain[T]) pushTail(v T) {
RETRY:
	tailChunk := atomicLoadChunk64[T](&inst.tailChunk) // load-1
	ti := atomic.AddInt64(&tailChunk.tailIdx, 1)       // incr-2
	if ti > inst.chunkSize {
		runtime.Gosched()
		//time.Sleep(time.Nanosecond * (time.Duration(ti%100) + 500))
		goto RETRY
	}

	if ti == inst.chunkSize {
		chunk := atomicSwapChunk64[T](&inst.memCache, nil)
		if chunk == nil {
			chunk = newChunk64[T](inst.chunkSize)

		} else {
			// load-1和incr-2之间可能会有其它协程多次的push/pop，tailChunk会被写满和读完。
			// 读完后，tailChunk会被consumer挂到memCache，进而会被producer再次写满时复用。
			// 此时incr-2会与store-3并发。
			// 如果incr-2先执行，则必会进入RETRY流程；
			// 如果store-3先执行，则相当于在最新的tailChunk上写入。
			atomic.StoreInt64(&chunk.tailIdx, 0) // store-3
		}

		tailChunk.next = chunk
		atomicStoreChunk64[T](&inst.tailChunk, chunk) // store-4
	}

	tailChunk.slot[ti-1].elt = v
	atomic.StoreInt64(&tailChunk.slot[ti-1].flag, 1)

	if inst.num.Add(1) < 1 {
		select {
		case inst.ch <- struct{}{}:
		default:
		}
	}
}

func (inst *mpscChain[T]) popHead() T {
	if inst.num.Add(-1) < 0 {
		<-inst.ch
	}

	headChunk := inst.headChunk
	pSlot := &headChunk.slot[headChunk.headIdx]
	for atomic.LoadInt64(&pSlot.flag) == 0 {
		//link.ProcYield(consumer_spin_cnt)
	}

	pSlot.flag = 0
	headChunk.headIdx++
	if headChunk.headIdx == inst.chunkSize {
		// headChunk.next此时必须重新load，才能够保证其一定不为空
		inst.headChunk = atomicLoadChunk64(&headChunk.next)

		ret := pSlot.elt
		headChunk.headIdx = 0
		headChunk.next = nil
		atomicSwapChunk64(&inst.memCache, headChunk)
		return ret
	}

	return pSlot.elt
}

func (inst *mpscRing[T]) pushTail(v T) {
	ti := atomic.AddInt64(&inst.tailIdx, 1) - 1
	for ti >= atomic.LoadInt64(&inst.headIdx)+inst.cap {
		runtime.Gosched()
	}

	pSlot := &inst.slot[ti&inst.mod]
	pSlot.elt = v
	atomic.StoreInt64(&pSlot.flag, 1)

	if inst.num.Add(1) < 1 {
		select {
		case inst.ch <- struct{}{}:
		default:
		}
	}
}

func (inst *mpscRing[T]) popHead() T {
	if inst.num.Add(-1) < 0 {
		<-inst.ch
	}

	pSlot := &inst.slot[inst.headIdx&inst.mod]
	var iter uint32
	for atomic.LoadInt64(&pSlot.flag) == 0 {
		iter++
		if iter > consumer_spin_cnt {
			//runtime.Gosched()
			time.Sleep(time.Nanosecond * time.Duration(500))

		} else {
			link.ProcYield(iter)
		}
	}

	v := pSlot.elt
	pSlot.flag = 0
	atomic.AddInt64(&inst.headIdx, 1)
	return v
}
