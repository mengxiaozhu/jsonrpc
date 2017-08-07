package jsonrpc

import (
	"context"
	"net"
	"net/rpc"
	"sync/atomic"
)

// *jsonrpc.Client
type Caller interface {
	Call(serviceMethod string, args interface{}, reply interface{}) error
}
type CallerFactory func(ctx context.Context) (Caller, error)

type PoolSender struct {
	clients []*LateInitCaller
	size    int
	times   uint64
}

func NewFixedPool(size int, factory CallerFactory) *PoolSender {
	cp := &PoolSender{
		size:    size,
		clients: make([]*LateInitCaller, size),
	}
	for i := 0; i < size; i++ {
		cp.clients[i] = &LateInitCaller{factory: factory}
	}
	return cp
}

func (c *PoolSender) Send(method string, ctx context.Context, v interface{}, resp interface{}) error {
	delay := c.clients[atomic.AddUint64(&c.times, 1)%uint64(c.size)]
	client, err := delay.Get(ctx)
	if err != nil {
		return err
	}
	err = client.Caller.Call(method, v, resp)
	if err != nil {
		if _, ok := err.(*net.OpError); ok || err == rpc.ErrShutdown {
			delay.Clear(client.Version)
		}
	}
	return err
}
