// Copyright (c) 2020 Andy Pan
// Copyright (c) 2017 Max Riveiro
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

//go:build linux || freebsd || dragonfly || darwin

package socket

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

var listenerBacklogMaxSize = maxListenerBacklog()

func TCPSocket(addr *net.TCPAddr, passive bool, sockOpts ...Option) (fd int, err error) {
	var (
		family   int
		ipv6only bool
		sa       unix.Sockaddr
	)

	if addr.IP.To4() != nil {
		family = unix.AF_INET
		sa, err = ipToSockaddr(family, addr.IP, addr.Port, "")

	} else if addr.IP.To16() != nil {
		family = unix.AF_INET6
		ipv6only = true
		sa, err = ipToSockaddr(family, addr.IP, addr.Port, addr.Zone)

	} else {
		family = unix.AF_INET6
	}

	if err != nil {
		return
	}

	if fd, err = Socket(family, unix.SOCK_STREAM, unix.IPPROTO_TCP); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}

	defer func() {
		// ignore EINPROGRESS for non-blocking socket connect, should be processed by caller
		if err != nil {
			if err, ok := err.(*os.SyscallError); ok && err.Err == unix.EINPROGRESS {
				return
			}
			_ = unix.Close(fd)
		}
	}()

	if family == unix.AF_INET6 && ipv6only {
		if err = SetIPv6Only(fd, 1); err != nil {
			return
		}
	}

	for _, sockOpt := range sockOpts {
		if err = sockOpt.SetSockOpt(fd, sockOpt.Opt); err != nil {
			return
		}
	}

	if passive {
		if err = os.NewSyscallError("bind", unix.Bind(fd, sa)); err != nil {
			return
		}
		// Set backlog size to the maximum.
		err = os.NewSyscallError("listen", unix.Listen(fd, listenerBacklogMaxSize))
	} else {
		err = os.NewSyscallError("connect", unix.Connect(fd, sa))
	}

	return
}
