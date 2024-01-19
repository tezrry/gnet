package netpoller

import (
	"context"
	"net"

	"github.com/panjf2000/gnet/v2/container/queue"
)

func newReactor(network Network, addr net.Addr, config *Config) (*_Reactor, error) {

	return nil, nil
}

type _Event struct {
	conn *Connection
	data []byte
}

type _Reactor struct {
	ctx    context.Context
	chErr  chan error
	rQueue queue.IQueue[_Event]
	wQueue queue.IQueue[_Event]
	nfd    int
}

func (inst *_Reactor) run(ctx context.Context, chErr chan error) {

}
