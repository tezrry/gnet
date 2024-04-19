// created by cgo -cdefs and then converted to Go
// cgo -cdefs defs2_linux.go

package evtpoll

type epollevent struct {
	events uint32
	data   [8]byte // to match amd64
}
