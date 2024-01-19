package gopool

import (
	"context"
	"sync"
)

var taskPool = sync.Pool{New: func() any { return new(_Task) }}
var taskFuturePool = sync.Pool{New: func() any { return new(_TaskFuture) }}

func newTask(ctx context.Context, f TaskFunc, param ...any) *_Task {
	inst := taskPool.Get().(*_Task)
	inst.ctx = ctx
	inst.f = f
	inst.param = param
	return inst
}

func freeTask(task *_Task) {
	task.ctx = nil
	task.f = nil
	task.param = nil
	taskPool.Put(task)
}

type _ITask interface {
	run()
}

type _Task struct {
	ctx   context.Context
	f     TaskFunc
	param []any
}

type _TaskFuture struct {
	ctx   context.Context
	f     TaskFutureFunc
	param []any
	chRsp chan any
}

func (inst *_Task) run() {
	if inst.ctx.Err() != nil {
		return
	}

	inst.f(inst.ctx, inst.param...)
}

func (inst *_TaskFuture) run() {
	if inst.ctx.Err() != nil {
		return
	}

	rsp, err := inst.f(inst.ctx, inst.param...)
	if err != nil {
		inst.chRsp <- err
		return
	}

	inst.chRsp <- rsp
}

func (inst *_TaskFuture) clear() {

}
