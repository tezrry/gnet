// Copyright (c) 2021 Andy Pan
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux

package evtpoll

import (
	"os"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/panjf2000/gnet/v2/internal/queue"
	"github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
)

type IOEvent = uint32

const (
	// InitPollEventsCap represents the initial capacity of poller event-list.
	InitPollEventsCap = 128
	// MaxPollEventsCap is the maximum limitation of events that the poller can process.
	MaxPollEventsCap = 1024
	// MinPollEventsCap is the minimum limitation of events that the poller can process.
	MinPollEventsCap = 32
	// MaxAsyncTasksAtOneTime is the maximum amount of asynchronous tasks that the event-loop will process at one time.
	MaxAsyncTasksAtOneTime = 256
	// ErrEvents represents exceptional events that are not read/write, like socket being closed,
	// reading/writing from/to a closed socket, etc.
	ErrEvents = unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
	// OutEvents combines EPOLLOUT event and some exceptional events.
	OutEvents = ErrEvents | unix.EPOLLOUT
	// InEvents combines EPOLLIN/EPOLLPRI events and some exceptional events.
	InEvents = ErrEvents | unix.EPOLLIN | unix.EPOLLPRI
)

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = unix.EAGAIN
	errEINVAL error = unix.EINVAL
	errENOENT error = unix.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e unix.Errno) error {
	switch e {
	case unix.EAGAIN:
		return errEAGAIN
	case unix.EINVAL:
		return errEINVAL
	case unix.ENOENT:
		return errENOENT
	}
	return e
}

var zero uintptr

type eventList struct {
	size   int
	events []epollevent
}

type Poller struct {
	fd int
}

func NewPoller() (poller *Poller, err error) {
	poller = new(Poller)
	if poller.fd, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}

	return
}

// Close closes the poller.
func (p *Poller) Close() error {
	if err := os.NewSyscallError("close", unix.Close(p.fd)); err != nil {
		return err
	}

	return nil
}

// Polling blocks the current goroutine, waiting for network-events.
func (p *Poller) Polling() error {
	el := &eventList{InitPollEventsCap, make([]epollevent, InitPollEventsCap)}
	var doChores bool

	msec := -1
	for {
		n, err := epollWait(p.fd, el.events, msec)
		if n == 0 || (n < 0 && err == unix.EINTR) {
			msec = -1
			runtime.Gosched()
			continue
		} else if err != nil {
			logging.Errorf("error occurs in epoll: %v", os.NewSyscallError("epoll_wait", err))
			return err
		}
		msec = 0

		for i := 0; i < n; i++ {
			ev := &el.events[i]
			pollAttachment := *(**PollAttachment)(unsafe.Pointer(&ev.data))
			if pollAttachment.FD != p.epa.FD {
				switch err = pollAttachment.Callback(pollAttachment.FD, ev.events); err {
				case nil:
				case errors.ErrAcceptSocket, errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in event-loop: %v", err)
				}
			} else { // poller is awakened to run tasks in queues.
				doChores = true
				_, _ = unix.Read(p.epa.FD, p.efdBuf)
			}
		}

		if doChores {
			doChores = false
			task := p.urgentAsyncTaskQueue.Dequeue()
			for ; task != nil; task = p.urgentAsyncTaskQueue.Dequeue() {
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			for i := 0; i < MaxAsyncTasksAtOneTime; i++ {
				if task = p.asyncTaskQueue.Dequeue(); task == nil {
					break
				}
				switch err = task.Run(task.Arg); err {
				case nil:
				case errors.ErrEngineShutdown:
					return err
				default:
					logging.Warnf("error occurs in user-defined function, %v", err)
				}
				queue.PutTask(task)
			}
			atomic.StoreInt32(&p.wakeupCall, 0)
			if (!p.asyncTaskQueue.IsEmpty() || !p.urgentAsyncTaskQueue.IsEmpty()) && atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
				switch _, err = unix.Write(p.epa.FD, b); err {
				case nil, unix.EAGAIN:
				default:
					doChores = true
				}
			}
		}

		if n == el.size {
			el.expand()
		} else if n < el.size>>1 {
			el.shrink()
		}
	}
}

func epollCtl(epfd int, op int, fd int, event *epollevent) error {
	_, _, errno := unix.RawSyscall6(unix.SYS_EPOLL_CTL, uintptr(epfd), uintptr(op), uintptr(fd), uintptr(unsafe.Pointer(event)), 0, 0)
	if errno != 0 {
		return errnoErr(errno)
	}
	return nil
}

// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
func (p *Poller) AddReadWrite(pa *PollAttachment) error {
	var ev epollevent
	ev.events = readWriteEvents
	*(**PollAttachment)(unsafe.Pointer(&ev.data)) = pa
	return os.NewSyscallError("epoll_ctl add", epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(pa *PollAttachment) error {
	var ev epollevent
	ev.events = readEvents
	*(**PollAttachment)(unsafe.Pointer(&ev.data)) = pa
	return os.NewSyscallError("epoll_ctl add", epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// AddWrite registers the given file-descriptor with writable event to the poller.
func (p *Poller) AddWrite(pa *PollAttachment) error {
	var ev epollevent
	ev.events = writeEvents
	*(**PollAttachment)(unsafe.Pointer(&ev.data)) = pa
	return os.NewSyscallError("epoll_ctl add", epollCtl(p.fd, unix.EPOLL_CTL_ADD, pa.FD, &ev))
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *Poller) ModRead(pa *PollAttachment) error {
	var ev epollevent
	ev.events = readEvents
	*(**PollAttachment)(unsafe.Pointer(&ev.data)) = pa
	return os.NewSyscallError("epoll_ctl mod", epollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &ev))
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *Poller) ModReadWrite(pa *PollAttachment) error {
	var ev epollevent
	ev.events = readWriteEvents
	*(**PollAttachment)(unsafe.Pointer(&ev.data)) = pa
	return os.NewSyscallError("epoll_ctl mod", epollCtl(p.fd, unix.EPOLL_CTL_MOD, pa.FD, &ev))
}

// Delete removes the given file-descriptor from the poller.
func (p *Poller) Delete(fd int) error {
	return os.NewSyscallError("epoll_ctl del", epollCtl(p.fd, unix.EPOLL_CTL_DEL, fd, nil))
}
