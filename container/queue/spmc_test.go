package queue

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func Benchmark_spmcChain_race(b *testing.B) {
	nConsumer := 2
	num := 1024000
	capacity := int64(256)

	benchmark_spmcChain(b, nConsumer, num, capacity)
}

func Benchmark_spmcRing_race(b *testing.B) {
	nConsumer := 4
	num := 1024000
	capacity := int64(256)

	benchmark_spmcRing(b, nConsumer, num, capacity)
}

func Benchmark_spmcRing_16_1024(b *testing.B) {
	nConsumer := 16
	num := 1024 * 1024
	capacity := int64(1024)

	benchmark_spmcRing(b, nConsumer, num, capacity)
}

func Benchmark_spmcChannel_16_1024(b *testing.B) {
	nConsumer := 16
	num := 1024 * 1020
	capacity := 1024

	benchmark_spmcChannel(b, nConsumer, num, capacity)
}

func Benchmark_spmcRingConsumer(b *testing.B) {
	capacity := int64(1024 * 1024)
	nConsumer, num := 1, int(capacity)
	for nConsumer <= num {
		name := fmt.Sprintf("%d", nConsumer)
		b.Run(name, func(b *testing.B) {
			benchmark_spmcRing(b, nConsumer, num, capacity)
		})

		nConsumer <<= 1
	}
}

func Benchmark_spmcRingCapacity(b *testing.B) {
	capacity := int64(1)
	nConsumer, num := 16, 1024*1024
	for capacity <= int64(num) {
		name := fmt.Sprintf("%d", capacity)
		b.Run(name, func(b *testing.B) {
			benchmark_spmcRing(b, nConsumer, num, capacity)
		})

		capacity <<= 1
	}
}

func Benchmark_spmcChannelConsumer(b *testing.B) {
	capacity := 1024 * 1024
	nConsumer, num := 1, capacity
	for nConsumer <= num {
		name := fmt.Sprintf("%d", nConsumer)
		b.Run(name, func(b *testing.B) {
			benchmark_spmcChannel(b, nConsumer, num, capacity)
		})

		nConsumer <<= 1
	}
}

func benchmark_spmcChain(b *testing.B, nConsumer, num int, capacity int64) {
	qList := make([]*spmcChain[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newChain_spmc[uint64](capacity)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(num * b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for n := 0; n < nConsumer; n++ {
			go func(i int) {
				for {
					v := qList[i].popHead()
					data[v].Add(1)
					wg.Done()
				}
			}(i)
		}

		go func(i int) {
			for j := 0; j < num; j++ {
				qList[i].pushTail(uint64(j))
			}
		}(i)
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.GreaterOrEqual(b, int64(0), qList[i].num.Load())
	}
	for i := 0; i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func benchmark_spmcRing(b *testing.B, nConsumer, num int, capacity int64) {
	qList := make([]*spmcRing[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newRing_spmc[uint64](capacity)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(num * b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for n := 0; n < nConsumer; n++ {
			go func(i int) {
				for {
					v := qList[i].popHead()
					data[v].Add(1)
					wg.Done()
				}
			}(i)
		}

		go func(i int) {
			for j := 0; j < num; j++ {
				qList[i].pushTail(uint64(j))
			}
		}(i)
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.GreaterOrEqual(b, int64(0), qList[i].num.Load())
	}
	//for i := 0; i < num; i++ {
	//	require.Equal(b, uint64(b.N), data[i].Load())
	//}
}

func benchmark_spmcChannel(b *testing.B, nConsumer, num, capacity int) {
	qList := make([]chan uint64, b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = make(chan uint64, capacity)
	}

	data := make([]atomic.Uint64, num)
	var wg sync.WaitGroup
	wg.Add(num * b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for n := 0; n < nConsumer; n++ {
			go func(i int) {
				for {
					v := <-qList[i]
					data[v].Add(1)
					wg.Done()
				}
			}(i)
		}

		go func(i int) {
			for j := 0; j < num; j++ {
				qList[i] <- uint64(j)
			}
		}(i)
	}

	wg.Wait()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, 0, len(qList[i]))
	}
	for i := 0; i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}
