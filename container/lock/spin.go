package lock

import (
	"sync/atomic"

	"github.com/panjf2000/gnet/v2/link"
)

type SpinLock int32

func (lk *SpinLock) Lock() {
	if atomic.CompareAndSwapInt32((*int32)(lk), 0, 1) {
		return
	}

	for !atomic.CompareAndSwapInt32((*int32)(lk), 0, 1) {
		r := link.FastRand() % 100
		if r < 30 {
			r = 30
		}
		//r := uint32(1000000)
		link.ProcYield(r)
		//println(time.Now().Unix())
	}
	//backoff := 1

}

func (lk *SpinLock) Unlock() {
	atomic.StoreInt32((*int32)(lk), 0)
}

func (lk *SpinLock) TryLock() {

}
