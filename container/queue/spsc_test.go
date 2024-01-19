package queue

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func Benchmark_spsc_all(b *testing.B) {
	num := uint32(1024 * 1024)
	capacity := uint32(1)
	for capacity <= num {
		name := fmt.Sprintf("%d", capacity)
		b.Log(name)
		b.Run("channel:"+name, func(b *testing.B) {
			bench_spscChannel(b, capacity, num)
		})
		b.Run("chain:"+name, func(b *testing.B) {
			bench_spscChain(b, capacity, num)
		})
		b.Run("ring:"+name, func(b *testing.B) {
			bench_spscRing(b, uint64(capacity), uint64(num))
		})

		capacity <<= 1
	}
}

func Benchmark_spscChain_1024x1024(b *testing.B) {
	nTotal := uint32(1024 * 1024)
	bench_spscChain(b, nTotal, nTotal)
}

func Benchmark_spscChannel_1024x1024(b *testing.B) {
	nTotal := uint32(1024 * 1024)
	bench_spscChannel(b, nTotal, nTotal)
}

func Benchmark_spscRing_1024x1024(b *testing.B) {
	nTotal := uint64(1024 * 1024)
	bench_spscRing(b, nTotal, nTotal)
}

func bench_spscRing(b *testing.B, cap, num uint64) {
	qList := make([]*spscRing[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newRing_spsc[uint64](cap)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := uint64(0); j < num; j++ {
				v := qList[i].popHead()
				data[v].Add(1)
			}
			wg.Done()
		}(i)

		go func(i int) {
			for j := uint64(0); j < num; j++ {
				qList[i].pushTail(j)
			}
		}(i)
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, int64(0), qList[i].num.Load())
	}
	for i := uint64(0); i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func bench_spscChannel(b *testing.B, cap, num uint32) {
	qList := make([]chan uint64, b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = make(chan uint64, cap)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := uint32(0); j < num; j++ {
				v := <-qList[i]
				data[v].Add(1)
			}

			wg.Done()
		}(i)

		go func(i int) {
			for j := uint32(0); j < num; j++ {
				qList[i] <- uint64(j)
			}

		}(i)

	}
	wg.Wait()

	for i := 0; i < b.N; i++ {
		require.Equal(b, 0, len(qList[i]))
	}
	for i := uint32(0); i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func bench_spscChain(b *testing.B, cap, num uint32) {
	qList := make([]*spscChain[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newChain_spsc[uint64](cap)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := uint32(0); j < num; j++ {
				v := qList[i].popHead()
				data[v].Add(1)
			}
			wg.Done()
		}(i)

		go func(i int) {
			for j := uint32(0); j < num; j++ {
				qList[i].pushTail(uint64(j))
			}
		}(i)
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, int64(0), qList[i].safeGuard.Load())
	}
	for i := uint32(0); i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}
