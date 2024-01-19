package link

import (
	"testing"
)

func TestAtomicLoadU64(t *testing.T) {
	var u64 = uint64(1)
	u := AtomicLoadUint64(&u64)
	t.Log(u)

	u = AtomicLoadAcqUint64(&u64)
	t.Log(u)
}
