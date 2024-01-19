package link

import _ "unsafe"

// Event types in the trace, args are given in square brackets.
const (
	traceEvGoBlock = 20 // goroutine blocks [timestamp, stack]
)

type mutex struct {
	// Futex-based impl treats it as uint32 key,
	// while sema-based impl as M* waitm.
	// Used to be a union, but unions break precise GC.
	key uintptr
}

//go:linkname lock runtime.lock
func lock(l *mutex)

//go:linkname unlock runtime.unlock
func unlock(l *mutex)

//go:linkname goparkunlock runtime.goparkunlock
func goparkunlock(lock *mutex, reason string, traceEv byte, traceskip int)

//go:linkname ProcYield runtime.procyield
func ProcYield(cycles uint32)

//go:linkname FastRand runtime.fastrand
func FastRand() uint32

//go:linkname AtomicLoadAcqUint64 runtime/internal/atomic.LoadAcq64
func AtomicLoadAcqUint64(addr *uint64) uint64

//go:linkname AtomicStoreRelUint64 runtime/internal/atomic.StoreRel64
func AtomicStoreRelUint64(addr *uint64, val uint64)

//go:linkname SemaAcquire runtime.semacquire
func SemaAcquire(s *uint32)

//go:linkname SemaRelease runtime.semrelease1
func SemaRelease(s *uint32, handoff bool, skipframes int)

func AtomicLoadAcqPtr() {

}
