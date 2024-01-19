package queue_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/panjf2000/gnet/v2/internal/queue"
	"github.com/stretchr/testify/require"
)

func TestLockFreeQueue(t *testing.T) {
	const taskNum = 10000
	q := queue.NewLockFreeQueue()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		for i := 0; i < taskNum; i++ {
			task := &queue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < taskNum; i++ {
			task := &queue.Task{}
			q.Enqueue(task)
		}
		wg.Done()
	}()

	var counter int32
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&counter) == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	go func() {
		for {
			task := q.Dequeue()
			if task != nil {
				atomic.AddInt32(&counter, 1)
			}
			if task == nil && atomic.LoadInt32(&counter) == 2*taskNum {
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()

	t.Logf("sent and received all %d tasks", 2*taskNum)
}

func Benchmark_mpsc_queue(b *testing.B) {
	b.Run("channel", func(b *testing.B) {
		_mpsc_channel(b, 100, 100, 1000000)
	})
	b.Run("queue", func(b *testing.B) {
		_mpsc_queue(b, 100, 100)
	})
}

func _mpsc_queue(b *testing.B, nProducer, num int) {
	nTotal := num * nProducer
	qList := make([]queue.AsyncTaskQueue, b.N)
	for i := 0; i < b.N; i++ {
		qList[i] = queue.NewLockFreeQueue()
	}

	data := make([]atomic.Uint64, nTotal)
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(i int) {
			for j := 0; j < nTotal; j++ {
				var t *queue.Task
				for t == nil {
					t = qList[i].Dequeue()
				}
				v := t.Arg.(uint64)
				data[v].Add(1)
			}
			wg.Done()
		}(i)

		for n := 0; n < nProducer; n++ {
			go func(i int, n int) {
				base := n * num
				for j := 0; j < num; j++ {
					qList[i].Enqueue(&queue.Task{Arg: uint64(base + j)})
				}
			}(i, n)
		}
	}
	wg.Wait()

	b.StopTimer()
	for i := 0; i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}

func _mpsc_channel(b *testing.B, nProducer, num, capacity int) {
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
	for i := 0; i < num; i++ {
		require.Equal(b, uint64(b.N), data[i].Load())
	}
}
