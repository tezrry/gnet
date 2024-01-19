package queue

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func Benchmark_mpscChannelCommon(b *testing.B) {
	benchmark_mpscChannel(b, 16, 65536, 1024)
}
func Benchmark_mpscChainCommon(b *testing.B) {
	benchmark_mpscChain(b, 16, 65536, 1024)
}

func Benchmark_mpscRingCommon(b *testing.B) {
	benchmark_mpscRing(b, 16, 65536, 1024)
}
func Benchmark_mpscChannel(b *testing.B) {
	capacity := 1024 * 1024
	nProducer, num := 1, capacity
	for nProducer <= capacity {
		name := fmt.Sprintf("%dx%d", nProducer, num)
		b.Run(name, func(b *testing.B) {
			benchmark_mpscChannel(b, nProducer, num, capacity)
		})

		nProducer <<= 1
		num = capacity / nProducer
	}
}
func Benchmark_mpscChainAll(b *testing.B) {
	capacity := int64(1024 * 1024)
	nProducer, num := int64(1), capacity
	for nProducer <= capacity {
		name := fmt.Sprintf("%dx%d", nProducer, num)
		b.Run(name, func(b *testing.B) {
			benchmark_mpscChain(b, nProducer, num, capacity)
		})

		nProducer <<= 1
		num = capacity / nProducer
	}
}
func Benchmark_mpscRingAll(b *testing.B) {
	capacity := int64(1024 * 1024)
	nProducer, num := int64(1), capacity
	for nProducer <= capacity {
		name := fmt.Sprintf("%dx%d", nProducer, num)
		b.Run(name, func(b *testing.B) {
			benchmark_mpscRing(b, nProducer, num, capacity)
		})

		nProducer <<= 1
		num = capacity / nProducer
	}
}
func Benchmark_mpscAll(b *testing.B) {
	capacity := int64(1024 * 1024)
	nProducer, num := int64(1), capacity
	for nProducer <= capacity {
		name := fmt.Sprintf("%dx%d", nProducer, num)
		b.Log(name)
		b.Run("channel:"+name, func(b *testing.B) {
			benchmark_mpscChannel(b, int(nProducer), int(num), int(capacity))
		})
		b.Run("chain:"+name, func(b *testing.B) {
			benchmark_mpscChain(b, nProducer, num, capacity)
		})
		b.Run("ring:"+name, func(b *testing.B) {
			benchmark_mpscRing(b, nProducer, num, capacity)
		})

		nProducer <<= 1
		num = capacity / nProducer
	}
}
func Benchmark_mpsc_16x65536(b *testing.B) {
	nProducer := 16
	num := 65536
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		name := fmt.Sprintf("%d", capacity)
		b.Log("capacity: " + name)

		b.Run("channel:"+name, func(b *testing.B) {
			benchmark_mpscChannel(b, nProducer, num, capacity)
		})
		b.Run("chain:"+name, func(b *testing.B) {
			benchmark_mpscChain(b, int64(nProducer), int64(num), int64(capacity))
		})
		b.Run("ring:"+name, func(b *testing.B) {
			benchmark_mpscRing(b, int64(nProducer), int64(num), int64(capacity))
		})

		capacity <<= 1
	}

}

func Benchmark_mpscChannel_16x65536(b *testing.B) {
	nProducer := 16
	num := 65536
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscChannel(b, nProducer, num, capacity)
		})

		capacity <<= 1
	}
}
func Benchmark_mpscChain_16x65536(b *testing.B) {
	nProducer := 16
	num := 65536
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscChain(b, int64(nProducer), int64(num), int64(capacity))
		})

		capacity <<= 1
	}
}

func Benchmark_mpscRing_16x65536(b *testing.B) {
	nProducer := 16
	num := 65536
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscRing(b, int64(nProducer), int64(num), int64(capacity))
		})

		capacity <<= 1
	}
}

func Benchmark_mpscChannel_32x32768(b *testing.B) {
	nProducer := 32
	num := 32768
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscChannel(b, nProducer, num, capacity)
		})

		capacity <<= 1
	}
}
func Benchmark_mpscChain_32x32768(b *testing.B) {
	nProducer := 32
	num := 32768
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscChain(b, int64(nProducer), int64(num), int64(capacity))
		})

		capacity <<= 1
	}
}

func Benchmark_mpscRing_32x32768(b *testing.B) {
	nProducer := 32
	num := 32768
	nTotal := nProducer * num
	capacity := 1

	for capacity <= nTotal {
		b.Run(fmt.Sprintf("%d", capacity), func(b *testing.B) {
			benchmark_mpscRing(b, int64(nProducer), int64(num), int64(capacity))
		})

		capacity <<= 1
	}
}

func Benchmark_mpscChain_race_200x100_100(b *testing.B) {
	nProducer := 10
	num := 10
	capacity := 16

	benchmark_mpscChain(b, int64(nProducer), int64(num), int64(capacity))
}

func Benchmark_mpscRing_race_200x100_100(b *testing.B) {
	nProducer := 200
	num := 200
	capacity := 16

	benchmark_mpscRing(b, int64(nProducer), int64(num), int64(capacity))
}

func benchmark_mpscChain(b *testing.B, nProducer, num, capacity int64) {
	nTotal := num * nProducer
	qList := make([]*mpscChain[uint64], b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = newChain_mpsc[uint64](capacity)
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
	for i := int64(0); i < nTotal; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func benchmark_mpscRing(b *testing.B, nProducer, num, capacity int64) {
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
	for i := int64(0); i < nTotal; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func benchmark_mpscChannel(b *testing.B, nProducer, num, capacity int) {
	nTotal := num * nProducer
	qList := make([]chan uint64, b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = make(chan uint64, capacity)
	}

	data := make([]atomic.Uint64, nTotal)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := 0; j < nTotal; j++ {
				v := <-qList[i]
				data[v].Add(1)
			}
			wg.Done()
		}(i)

		for n := 0; n < nProducer; n++ {
			go func(i int, n int) {
				base := n * num
				for j := 0; j < num; j++ {
					qList[i] <- uint64(base + j)
				}
			}(i, n)
		}
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, 0, len(qList[i]))
	}
	for i := 0; i < nTotal; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}
