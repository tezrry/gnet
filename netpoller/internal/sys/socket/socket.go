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

//go:build linux || darwin

package socket

import (
	"net"
	"os"
	"syscall"

	"github.com/panjf2000/gnet/v2/pkg/errors"
	"golang.org/x/sys/unix"
)

type Option struct {
	Call  func(int, int) error
	Param int
}

// SetNoDelay controls whether the operating system should delay
// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
//
// The default is true (no delay), meaning that data is
// sent as soon as possible after writing.
func SetNoDelay(fd, noDelay int) error {
	return os.NewSyscallError("SetNoDelay", unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, noDelay))
}

// SetRecvBuffer sets the size of the operating system's
// receive buffer associated with the connection.
func SetRecvBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size)
}

// SetSendBuffer sets the size of the operating system's
// transmit buffer associated with the connection.
func SetSendBuffer(fd, size int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size)
}

// SetReuseport enables SO_REUSEPORT option on socket.
func SetReuseport(fd, reusePort int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, reusePort))
}

// SetReuseAddr enables SO_REUSEADDR option on socket.
func SetReuseAddr(fd, reuseAddr int) error {
	return os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, reuseAddr))
}

// SetIPv6Only restricts a IPv6 socket to only process IPv6 requests or both IPv4 and IPv6 requests.
func SetIPv6Only(fd, ipv6only int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, ipv6only)
}

// SetLinger sets the behavior of Close on a connection which still
// has data waiting to be sent or to be acknowledged.
//
// If sec < 0 (the default), the operating system finishes sending the
// data in the background.
//
// If sec == 0, the operating system discards any unsent or
// unacknowledged data.
//
// If sec > 0, the data is sent in the background as with sec < 0. On
// some operating systems after sec seconds have elapsed any remaining
// unsent data may be discarded.
func SetLinger(fd, sec int) error {
	var l unix.Linger
	if sec >= 0 {
		l.Onoff = 1
		l.Linger = int32(sec)
	} else {
		l.Onoff = 0
		l.Linger = 0
	}
	return unix.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &l)
}

// SetMulticastMembership returns with a socket option function based on the IP
// version. Returns nil when multicast membership cannot be applied.
func SetMulticastMembership(proto string, udpAddr *net.UDPAddr) func(int, int) error {
	udpVersion, err := determineUDPProto(proto, udpAddr)
	if err != nil {
		return nil
	}

	switch udpVersion {
	case "udp4":
		return func(fd int, ifIndex int) error {
			return SetIPv4MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	case "udp6":
		return func(fd int, ifIndex int) error {
			return SetIPv6MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	default:
		return nil
	}
}

// SetIPv4MulticastMembership joins fd to the specified multicast IPv4 address.
// ifIndex is the index of the interface where the multicast datagrams will be
// received. If ifIndex is 0 then the operating system will choose the default,
// it is usually needed when the host has multiple network interfaces configured.
func SetIPv4MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	// Multicast interfaces are selected by IP address on IPv4 (and by index on IPv6)
	ip, err := interfaceFirstIPv4Addr(ifIndex)
	if err != nil {
		return err
	}

	mreq := &unix.IPMreq{}
	copy(mreq.Multiaddr[:], mcast.To4())
	copy(mreq.Interface[:], ip.To4())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt", unix.SetsockoptInet4Addr(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, mreq.Interface)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt", unix.SetsockoptByte(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 0)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptIPMreq(fd, syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq))
}

// SetIPv6MulticastMembership joins fd to the specified multicast IPv6 address.
// ifIndex is the index of the interface where the multicast datagrams will be
// received. If ifIndex is 0 then the operating system will choose the default,
// it is usually needed when the host has multiple network interfaces configured.
func SetIPv6MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	mreq := &unix.IPv6Mreq{}
	mreq.Interface = uint32(ifIndex)
	copy(mreq.Multiaddr[:], mcast.To16())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_IF, ifIndex)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 0)); err != nil {
		return err
	}
	return os.NewSyscallError("setsockopt", unix.SetsockoptIPv6Mreq(fd, syscall.IPPROTO_IPV6, syscall.IPV6_JOIN_GROUP, mreq))
}

// interfaceFirstIPv4Addr returns the first IPv4 address of the interface.
func interfaceFirstIPv4Addr(ifIndex int) (net.IP, error) {
	if ifIndex == 0 {
		return net.IP([]byte{0, 0, 0, 0}), nil
	}
	iface, err := net.InterfaceByIndex(ifIndex)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}
		if ip.To4() != nil {
			return ip, nil
		}
	}
	return nil, errors.ErrNoIPv4AddressOnInterface
}

func ipToSockaddrInet4(ip net.IP, port int) (unix.SockaddrInet4, error) {
	if len(ip) == 0 {
		ip = net.IPv4zero
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return unix.SockaddrInet4{}, &net.AddrError{Err: "non-IPv4 address", Addr: ip.String()}
	}
	sa := unix.SockaddrInet4{Port: port}
	copy(sa.Addr[:], ip4)
	return sa, nil
}

func ipToSockaddrInet6(ip net.IP, port int, zone string) (unix.SockaddrInet6, error) {
	// In general, an IP wildcard address, which is either
	// "0.0.0.0" or "::", means the entire IP addressing
	// space. For some historical reason, it is used to
	// specify "any available address" on some operations
	// of IP node.
	//
	// When the IP node supports IPv4-mapped IPv6 address,
	// we allow a listener to listen to the wildcard
	// address of both IP addressing spaces by specifying
	// IPv6 wildcard address.
	if len(ip) == 0 || ip.Equal(net.IPv4zero) {
		ip = net.IPv6zero
	}
	// We accept any IPv6 address including IPv4-mapped
	// IPv6 address.
	ip6 := ip.To16()
	if ip6 == nil {
		return unix.SockaddrInet6{}, &net.AddrError{Err: "non-IPv6 address", Addr: ip.String()}
	}

	sa := unix.SockaddrInet6{Port: port}
	copy(sa.Addr[:], ip6)
	iface, err := net.InterfaceByName(zone)
	if err != nil {
		return sa, nil
	}
	sa.ZoneId = uint32(iface.Index)

	return sa, nil
}

func ipToSockaddr(family int, ip net.IP, port int, zone string) (unix.Sockaddr, error) {
	switch family {
	case syscall.AF_INET:
		sa, err := ipToSockaddrInet4(ip, port)
		if err != nil {
			return nil, err
		}
		return &sa, nil
	case syscall.AF_INET6:
		sa, err := ipToSockaddrInet6(ip, port, zone)
		if err != nil {
			return nil, err
		}
		return &sa, nil
	}
	return nil, &net.AddrError{Err: "invalid address family", Addr: ip.String()}
}

var ipv4InIPv6Prefix = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}

//// SockaddrToTCPOrUnixAddr converts a Sockaddr to a net.TCPAddr or net.UnixAddr.
//// Returns nil if conversion fails.
//func SockaddrToTCPOrUnixAddr(sa unix.Sockaddr) net.Addr {
//	switch sa := sa.(type) {
//	case *unix.SockaddrInet4:
//		ip := sockaddrInet4ToIP(sa)
//		return &net.TCPAddr{IP: ip, Port: sa.Port}
//	case *unix.SockaddrInet6:
//		ip, zone := sockaddrInet6ToIPAndZone(sa)
//		return &net.TCPAddr{IP: ip, Port: sa.Port, Zone: zone}
//	case *unix.SockaddrUnix:
//		return &net.UnixAddr{Name: sa.Name, Net: "unix"}
//	}
//	return nil
//}

//// SockaddrToUDPAddr converts a Sockaddr to a net.UDPAddr
//// Returns nil if conversion fails.
//func SockaddrToUDPAddr(sa unix.Sockaddr) net.Addr {
//	switch sa := sa.(type) {
//	case *unix.SockaddrInet4:
//		ip := sockaddrInet4ToIP(sa)
//		return &net.UDPAddr{IP: ip, Port: sa.Port}
//	case *unix.SockaddrInet6:
//		ip, zone := sockaddrInet6ToIPAndZone(sa)
//		return &net.UDPAddr{IP: ip, Port: sa.Port, Zone: zone}
//	}
//	return nil
//}
//
//// sockaddrInet4ToIPAndZone converts a SockaddrInet4 to a net.IP.
//// It returns nil if conversion fails.
//func sockaddrInet4ToIP(sa *unix.SockaddrInet4) net.IP {
//	ip := bsPool.Get(16)
//	// ipv4InIPv6Prefix
//	copy(ip[0:12], ipv4InIPv6Prefix)
//	copy(ip[12:16], sa.Addr[:])
//	return ip
//}
//
//// sockaddrInet6ToIPAndZone converts a SockaddrInet6 to a net.IP with IPv6 Zone.
//// It returns nil if conversion fails.
//func sockaddrInet6ToIPAndZone(sa *unix.SockaddrInet6) (net.IP, string) {
//	ip := bsPool.Get(16)
//	copy(ip, sa.Addr[:])
//	return ip, ip6ZoneToString(int(sa.ZoneId))
//}
//
//// ip6ZoneToString converts an IP6 Zone unix int to a net string
//// returns "" if zone is 0.
//func ip6ZoneToString(zone int) string {
//	if zone == 0 {
//		return ""
//	}
//	if ifi, err := net.InterfaceByIndex(zone); err == nil {
//		return ifi.Name
//	}
//	return int2decimal(uint(zone))
//}
//
//// Convert int to decimal string.
//func int2decimal(i uint) string {
//	if i == 0 {
//		return "0"
//	}
//
//	// Assemble decimal in reverse order.
//	b := bsPool.Get(32)
//	bp := len(b)
//	for ; i > 0; i /= 10 {
//		bp--
//		b[bp] = byte(i%10) + '0'
//	}
//	return toolkit.BytesToString(b[bp:])
//}
