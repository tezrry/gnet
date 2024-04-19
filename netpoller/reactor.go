package netpoller

import (
	"context"
	"fmt"
	"net"

	"github.com/panjf2000/gnet/v2/container/queue"
	"github.com/panjf2000/gnet/v2/netpoller/internal/sys/evtpoll"
	"github.com/panjf2000/gnet/v2/netpoller/internal/sys/socket"
)

type _Reactor struct {
	ctx     context.Context
	config  *Config
	network Network
	netAddr net.Addr
	chErr   chan error
	rQueue  queue.IQueue[_Event]
	wQueue  queue.IQueue[_Event]
	poller  *evtpoll.Poller
}

func newReactor(network Network, addr net.Addr, config *Config) (*_Reactor, error) {
	inst := &_Reactor{
		config:  config,
		network: network,
		netAddr: addr,
		rQueue:  nil,
		wQueue:  nil,
	}

	var err error
	inst.poller, err = evtpoll.NewPoller()
	if err != nil {
		return nil, err
	}

	return inst, nil
}

type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}

type _Event struct {
	conn *Connection
	data []byte
}

func (inst *_Reactor) run(ctx context.Context, chErr <-chan error) {
	sockOpts := make([]socket.Option, 0, 8)
	if inst.config.SocketSendBuffer > 0 {
		sockOpts = append(sockOpts, socket.Option{
			Call:  socket.SetSendBuffer,
			Param: inst.config.SocketSendBuffer,
		})
	}
	if inst.config.SocketRecvBuffer > 0 {
		sockOpts = append(sockOpts, socket.Option{
			Call:  socket.SetRecvBuffer,
			Param: inst.config.SocketRecvBuffer,
		})
	}

	switch inst.network {
	case NetworkTCP:
		sockOpts = append(sockOpts, socket.Option{
			Call:  socket.SetReuseAddr,
			Param: 1,
		})
		sockOpts = append(sockOpts, socket.Option{
			Call:  socket.SetReuseport,
			Param: 1,
		})
		if inst.config.TCPNoDelay {
			sockOpts = append(sockOpts, socket.Option{
				Call:  socket.SetNoDelay,
				Param: 1,
			})
		}

		fd, err := socket.TCPSocket(inst.netAddr.(*net.TCPAddr), true, sockOpts...)
		if err != nil {
			return
		}

		err := inst.poller.Start()
		if err != nil {
			return
		}

	case NetworkUDP:
		sockOpts = append(sockOpts, socket.Option{
			Call:  socket.SetReuseport,
			Param: 1,
		})

	case NetworkUNIX:
	default:
		panic(fmt.Errorf("non-support network type"))
	}

}
