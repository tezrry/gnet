package netpoller

import "net"

type Connection struct {
	rawConn net.Conn
}
