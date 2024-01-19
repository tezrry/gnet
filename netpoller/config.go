package netpoller

import "time"

type ConfigFunc func(c *Config)

type Config struct {
	ReactorNum    int
	HandlerMaxNum int

	Concurrency         int
	SingleThreadHandler bool

	// TCPKeepAlive sets up a duration for (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration

	// TCPNoDelay controls whether the operating system should delay
	// packet transmission in hopes of sending fewer packets (Nagle's algorithm).
	//
	// The default is true (no delay), meaning that data is sent
	// as soon as possible after a write operation.
	TCPNoDelay bool

	// SocketRecvBuffer sets the maximum socket receive buffer in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer in bytes.
	SocketSendBuffer int
}

func WithConfig(config *Config) ConfigFunc {
	return func(c *Config) {
		*c = *config
	}
}

func WithConcurrency(num int) ConfigFunc {
	return func(c *Config) {
		c.Concurrency = num
	}
}

func WithSingleThreadHandler(v bool) ConfigFunc {
	return func(c *Config) {
		c.SingleThreadHandler = v
	}
}
