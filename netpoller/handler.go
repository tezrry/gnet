package netpoller

type Handler interface {
	OnConnect(conn *Connection)
	OnRead()
	OnDisconnect()
}
