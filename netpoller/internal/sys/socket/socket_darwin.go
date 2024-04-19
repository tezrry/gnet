// Copyright (c) 2020 Andy Pan
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

//go:build darwin

package socket

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func Socket(family, soType, proto int) (fd int, err error) {
	syscall.ForkLock.RLock()
	if fd, err = unix.Socket(family, soType, proto); err == nil {
		unix.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()

	if err != nil {
		return
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		_ = unix.Close(fd)
	}

	return
}

// SetKeepAlivePeriod sets whether the operating system should send
// keep-alive messages on the connection and sets period between keep-alive's.
func SetKeepAlivePeriod(fd, secs int) error {
	if secs <= 0 {
		return errors.New("invalid time duration")
	}
	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, 1)); err != nil {
		return err
	}
	switch err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, secs)); err {
	case nil, unix.ENOPROTOOPT: // OS X 10.7 and earlier don't support this option
	default:
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPALIVE, secs))
}

func maxListenerBacklog() int {
	var (
		n   uint32
		err error
	)
	switch runtime.GOOS {
	case "darwin":
		n, err = unix.SysctlUint32("kern.ipc.somaxconn")
	case "freebsd":
		n, err = unix.SysctlUint32("kern.ipc.soacceptqueue")
	}
	if n == 0 || err != nil {
		return unix.SOMAXCONN
	}
	// FreeBSD stores the backlog in an uint16, as does Linux.
	// Assume the other BSDs do too. Truncate number to avoid wrapping.
	// See issue 5030.
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	return int(n)
}
