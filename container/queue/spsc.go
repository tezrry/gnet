package queue

import (
	"runtime"
	"sync/atomic"
	"unsafe"

	util_math "github.com/panjf2000/gnet/v2/util/math"
)

type spscChunk[T any] struct {
	slot []T
	next *spscChunk[T]
	prev *spscChunk[T]
}

type spscChain[T any] struct {
	safeGuard atomic.Int64
	headChunk *spscChunk[T]
	tailChunk *spscChunk[T]
	memCache  *spscChunk[T]
	headIdx   uint32
	tailIdx   uint32
	ch        chan struct{}
	chunkSize uint32
}

type spscRing[T any] struct {
	num     atomic.Int64
	headIdx uint64
	tailIdx uint64
	slot    []T
	cap     uint64
	mod     uint64
	ch      chan struct{}
}

func newChain_spsc[T any](chunkSize uint32) *spscChain[T] {
	//size := uint32(math.CeilToPowerOfTwo(uint64(chunkSize)))
	size := chunkSize
	chunk := newChunk_spsc[T](size)

	return &spscChain[T]{
		headChunk: chunk,
		tailChunk: chunk,
		chunkSize: size,
		ch:        make(chan struct{}, 1),
	}
}

func newChunk_spsc[T any](chunkSize uint32) *spscChunk[T] {
	return &spscChunk[T]{
		slot: make([]T, chunkSize),
	}
}

func newRing_spsc[T any](size uint64) *spscRing[T] {
	size = util_math.CeilToPowerOfTwo(size)
	return &spscRing[T]{
		slot: make([]T, size),
		cap:  size,
		mod:  size - 1,
		ch:   make(chan struct{}, 1),
	}
}
func atomicSwapChunkPtr_spsc[T any](addr **spscChunk[T], new *spscChunk[T]) *spscChunk[T] {
	return (*spscChunk[T])(atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(addr)), unsafe.Pointer(new)))
}

func atomicLoadChunkPtr_spsc[T any](addr **spscChunk[T]) *spscChunk[T] {
	return (*spscChunk[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(addr))))
}

func (inst *spscChain[T]) pushTail(v T) {
	inst.tailChunk.slot[inst.tailIdx] = v
	inst.tailIdx++
	if inst.tailIdx == inst.chunkSize {
		chunk := atomicSwapChunkPtr_spsc[T](&inst.memCache, nil)
		if chunk == nil {
			chunk = newChunk_spsc[T](inst.chunkSize)
		}
		chunk.prev = inst.tailChunk
		inst.tailChunk.next = chunk
		inst.tailChunk = chunk
		inst.tailIdx = 0
	}

	if inst.safeGuard.Add(1) < 1 {
		select {
		case inst.ch <- struct{}{}:
		default:
		}
	}
}

func (inst *spscChain[T]) len() int64 {
	v := inst.safeGuard.Load()
	if v < 0 {
		return 0
	}
	return v
}

func (inst *spscChain[T]) popHead() T {
	if inst.safeGuard.Add(-1) < 0 {
		<-inst.ch
	}

	chunk := inst.headChunk
	inst.headIdx++
	if inst.headIdx == inst.chunkSize {
		// chunk.next一定不为空，可以安全的赋值
		//inst.headChunk = atomicLoadChunkPtr_spsc[T](&chunk.next)
		inst.headChunk = chunk.next
		inst.headChunk.prev = nil

		ret := chunk.slot[inst.headIdx-1]
		inst.headIdx = 0

		chunk.next = nil
		atomicSwapChunkPtr_spsc[T](&inst.memCache, chunk)
		return ret
	}

	return chunk.slot[inst.headIdx-1]
}

func (inst *spscRing[T]) pushTail(v T) {
	for atomic.LoadUint64(&inst.headIdx)+inst.cap <= inst.tailIdx {
		runtime.Gosched()
	}

	inst.slot[inst.tailIdx&inst.mod] = v
	inst.tailIdx++

	if inst.num.Add(1) < 1 {
		select {
		case inst.ch <- struct{}{}:
		default:
		}
	}
}

func (inst *spscRing[T]) popHead() T {
	if inst.num.Add(-1) < 0 {
		<-inst.ch
	}

	v := inst.slot[inst.headIdx&inst.mod]
	atomic.AddUint64(&inst.headIdx, 1)
	return v
}
