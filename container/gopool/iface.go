package gopool

import "context"

type TaskFunc func(ctx context.Context, param ...interface{})
type TaskFutureFunc func(ctx context.Context, param ...interface{}) (interface{}, error)

type Pool interface {
	Schedule(ctx context.Context, task TaskFunc, param ...interface{})
	ScheduleFuture(ctx context.Context, chRsp chan interface{}, task TaskFutureFunc, param ...interface{})
}
