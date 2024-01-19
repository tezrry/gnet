package queue

type QueueType uint8

const (
	PMCM QueueType = 0
	PMCS QueueType = 1
	PSCM QueueType = 2
	PSCS QueueType = 3
	PMCMList
)

type IQueue[T any] interface {
	Enqueue(v T) bool
	Dequeue() T
}
