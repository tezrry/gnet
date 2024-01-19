package lock

import "testing"

func TestSpinLock(t *testing.T) {
	var lk SpinLock
	lk.Lock()
	lk.Lock()
	lk.Unlock()
}

func BenchmarkSpinLock(b *testing.B) {
	var lk SpinLock
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lk.Lock()
			lk.Unlock()
		}
	})
}
