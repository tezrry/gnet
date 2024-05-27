//go:build freebsd || dragonfly || netbsd || openbsd || darwin

package evtpoll

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	// InitPollEventsCap represents the initial capacity of poller event-list.
	InitPollEventsCap = 64
	// MaxPollEventsCap is the maximum limitation of events that the poller can process.
	MaxPollEventsCap = 512
	// MinPollEventsCap is the minimum limitation of events that the poller can process.
	MinPollEventsCap = 16
	// MaxAsyncTasksAtOneTime is the maximum amount of asynchronous tasks that the event-loop will process at one time.
	MaxAsyncTasksAtOneTime = 128
	// EVFilterWrite represents writeable events from sockets.
	EVFilterWrite = unix.EVFILT_WRITE
	// EVFilterRead represents readable events from sockets.
	EVFilterRead = unix.EVFILT_READ
	// EVFilterSock represents exceptional events that are not read/write, like socket being closed,
	// reading/writing from/to a closed socket, etc.
	EVFilterSock = -0xd
)

type Poller struct {
	efd    int
	wakeup int32
}

var initEvt = []syscall.Kevent_t{{
	Ident:  0,
	Filter: syscall.EVFILT_USER,
	Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
}}

var note = []syscall.Kevent_t{{
	Ident:  0,
	Filter: syscall.EVFILT_USER,
	Fflags: syscall.NOTE_TRIGGER,
}}

func NewPoller() (*Poller, error) {
	poller := new(Poller)
	var err error
	if poller.efd, err = syscall.Kqueue(); err != nil {
		return nil, os.NewSyscallError("kqueue", err)
	}

	if _, err = syscall.Kevent(poller.efd, initEvt, nil, nil); err != nil {
		_ = poller.Close()
		return nil, os.NewSyscallError("kevent add|clear", err)
	}

	return poller, nil
}

func (p *Poller) Start() error {
	evts := make([]syscall.Kevent_t, InitPollEventsCap)
	var ts syscall.Timespec
	var tsp *syscall.Timespec
	for {
		n, err := syscall.Kevent(p.efd, nil, evts, tsp)
		if n == 0 || (n < 0 && err == syscall.EINTR) {
			tsp = nil
			runtime.Gosched()
			continue

		} else if err != nil {
			return os.NewSyscallError("kevent wait", err)
		}

		tsp = &ts
		var evFilter int16
		for i := 0; i < n; i++ {
			ev := &evts[i]
			if ev.Ident != 0 {
				evFilter = ev.Filter
				if (ev.Flags&syscall.EV_EOF != 0) || (ev.Flags&syscall.EV_ERROR != 0) {
					evFilter = EVFilterSock
				}

				//pollAttachment := (*PollAttachment)(unsafe.Pointer(ev.Udata))
				//switch err = pollAttachment.Callback(int(ev.Ident), evFilter); err {
				//case nil:
				//case errors.ErrAcceptSocket, errors.ErrEngineShutdown:
				//	return err
				//default:
				//	logging.Warnf("error occurs in event-loop: %v", err)
				//}
			} else { // poller is awakened to run tasks in queues.
			}
		}

		println(evFilter)
	}
}

func (p *Poller) Close() error {
	return os.NewSyscallError("close", unix.Close(p.efd))
}

// AddRead registers the given file-descriptor with readable event to the poller.
func (p *Poller) AddRead(fd int, data unsafe.Pointer) error {
	var evs [1]unix.Kevent_t
	evs[0].Ident = uint64(fd)
	evs[0].Flags = unix.EV_ADD
	evs[0].Filter = unix.EVFILT_READ
	evs[0].Udata = (*byte)(data)
	_, err := unix.Kevent(p.efd, evs[:], nil, nil)
	return os.NewSyscallError("kevent add", err)
}

//// UrgentTrigger puts task into urgentAsyncTaskQueue and wakes up the poller which is waiting for network-events,
//// then the poller will get tasks from urgentAsyncTaskQueue and run them.
////
//// Note that urgentAsyncTaskQueue is a queue with high-priority and its size is expected to be small,
//// so only those urgent tasks should be put into this queue.
//func (p *Poller) UrgentTrigger(fn queue.TaskFunc, arg interface{}) (err error) {
//	task := queue.GetTask()
//	task.Run, task.Arg = fn, arg
//	p.urgentAsyncTaskQueue.Enqueue(task)
//	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
//		if _, err = unix.Kevent(p.fd, note, nil, nil); err == unix.EAGAIN {
//			err = nil
//		}
//	}
//	return os.NewSyscallError("kevent trigger", err)
//}
//
//// Trigger is like UrgentTrigger but it puts task into asyncTaskQueue,
//// call this method when the task is not so urgent, for instance writing data back to the peer.
////
//// Note that asyncTaskQueue is a queue with low-priority whose size may grow large and tasks in it may backlog.
//func (p *Poller) Trigger(fn queue.TaskFunc, arg interface{}) (err error) {
//	task := queue.GetTask()
//	task.Run, task.Arg = fn, arg
//	p.asyncTaskQueue.Enqueue(task)
//	if atomic.CompareAndSwapInt32(&p.wakeupCall, 0, 1) {
//		if _, err = unix.Kevent(p.fd, note, nil, nil); err == unix.EAGAIN {
//			err = nil
//		}
//	}
//	return os.NewSyscallError("kevent trigger", err)
//}
//
//// AddReadWrite registers the given file-descriptor with readable and writable events to the poller.
//func (p *Poller) AddReadWrite(pa *PollAttachment) error {
//	var evs [2]unix.Kevent_t
//	evs[0].Ident = uint64(pa.FD)
//	evs[0].Flags = unix.EV_ADD
//	evs[0].Filter = unix.EVFILT_READ
//	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
//	evs[1] = evs[0]
//	evs[1].Filter = unix.EVFILT_WRITE
//	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
//	return os.NewSyscallError("kevent add", err)
//}
//

//
//// AddWrite registers the given file-descriptor with writable event to the poller.
//func (p *Poller) AddWrite(pa *PollAttachment) error {
//	var evs [1]unix.Kevent_t
//	evs[0].Ident = uint64(pa.FD)
//	evs[0].Flags = unix.EV_ADD
//	evs[0].Filter = unix.EVFILT_WRITE
//	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
//	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
//	return os.NewSyscallError("kevent add", err)
//}
//
//// ModRead renews the given file-descriptor with readable event in the poller.
//func (p *Poller) ModRead(pa *PollAttachment) error {
//	var evs [1]unix.Kevent_t
//	evs[0].Ident = uint64(pa.FD)
//	evs[0].Flags = unix.EV_DELETE
//	evs[0].Filter = unix.EVFILT_WRITE
//	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
//	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
//	return os.NewSyscallError("kevent delete", err)
//}
//
//// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
//func (p *Poller) ModReadWrite(pa *PollAttachment) error {
//	var evs [1]unix.Kevent_t
//	evs[0].Ident = uint64(pa.FD)
//	evs[0].Flags = unix.EV_ADD
//	evs[0].Filter = unix.EVFILT_WRITE
//	evs[0].Udata = (*byte)(unsafe.Pointer(pa))
//	_, err := unix.Kevent(p.fd, evs[:], nil, nil)
//	return os.NewSyscallError("kevent add", err)
//}
//
//// Delete removes the given file-descriptor from the poller.
//func (p *Poller) Delete(_ int) error {
//	return nil
//}
