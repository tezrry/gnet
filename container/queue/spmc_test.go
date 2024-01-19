package queue

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func Benchmark_spmcChain_race(b *testing.B) {
	nConsumer := 16
	num := 10240
	capacity := int64(256)

	benchmark_spmcChain(b, nConsumer, num, capacity)
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
	//for i := 0; i < b.N; i++ {
	//	require.Equal(b, int64(0), qList[i].num.Load())
	//}
	for i := 0; i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func benchmark_spmcRing(b *testing.B, nProducer, num, capacity int64) {
	nTotal := num * nProducer
	qList := make([]*mpscRing[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newRing_mpsc[uint64](capacity)
	}

	data := make([]atomic.Uint64, nTotal)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := int64(0); j < nTotal; j++ {
				v := qList[i].popHead()
				data[v].Add(1)
			}
			wg.Done()
		}(i)

		for n := int64(0); n < nProducer; n++ {
			go func(i int, n int64) {
				base := n * num
				for j := int64(0); j < num; j++ {
					qList[i].pushTail(uint64(base + j))
				}
			}(i, n)
		}
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, int64(0), qList[i].num.Load())
	}
	for i := int64(0); i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
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
