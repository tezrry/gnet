package netpoller

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
)

type Network uint8

const (
	NetworkUnknown Network = 0
	NetworkTCP     Network = 1
	NetworkUDP     Network = 2
	NetworkUNIX    Network = 3
)

func CreateService(proto string, config ...ConfigFunc) (*Service, error) {
	network, addr, err := parseProtocol(proto)
	if err != nil {
		return nil, err
	}

	inst := &Service{
		config: Config{
			Concurrency:         0,
			SingleThreadHandler: false,
			TCPKeepAlive:        0,
			TCPNoDelay:          false,
			SocketRecvBuffer:    0,
			SocketSendBuffer:    0,
		},
		network: network,
		netAddr: addr,
	}

	for _, cf := range config {
		cf(&inst.config)
	}

	if inst.config.Concurrency == 0 {
		inst.config.Concurrency = runtime.NumCPU()
	}

	return inst, nil
}

func parseProtocol(protocol string) (Network, net.Addr, error) {
	network, address := "tcp", ""
	pair := strings.Split(strings.ToLower(protocol), "://")
	switch len(pair) {
	case 1:
		address = pair[0]
		break

	case 2:
		network = pair[0]
		address = pair[1]

	default:
		return NetworkUnknown, nil, fmt.Errorf("invalid protocol %s", protocol)
	}

	if strings.HasPrefix(network, "tcp") {
		ret, err := net.ResolveTCPAddr(network, address)
		if err != nil {
			return NetworkUnknown, nil, fmt.Errorf("not support tcp protocol %s", protocol)
		}

		return NetworkTCP, ret, nil
	}

	if strings.HasPrefix(network, "udp") {
		ret, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			return NetworkUnknown, nil, fmt.Errorf("not support udp protocol %s", protocol)
		}

		return NetworkUDP, ret, nil
	}

	if strings.HasPrefix(network, "unix") {
		ret, err := net.ResolveUnixAddr(network, address)
		if err != nil {
			return NetworkUnknown, nil, fmt.Errorf("not support unix protocol %s", protocol)
		}

		return NetworkUNIX, ret, nil
	}

	return NetworkUnknown, nil, fmt.Errorf("not support protocol %s", protocol)
}

type Service struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	chErr   chan error
	config  Config
	network Network
	netAddr net.Addr
	reactor []*_Reactor
}

func (inst *Service) Run(ctx context.Context) chan error {
	inst.ctx, inst.cancel = context.WithCancel(ctx)
	inst.chErr = make(chan error, inst.config.ReactorNum)
	for i := 0; i < inst.config.ReactorNum; i++ {

	}

	return inst.chErr
}

func (inst *Service) Stop() *sync.WaitGroup {
	return &inst.wg
}

type SubPoller interface {
	Run() (context.CancelFunc, error)
}
