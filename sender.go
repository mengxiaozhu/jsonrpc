package jsonrpc

import (
	"context"
	"sync/atomic"
)

type PoolSender struct {
	callers []*LateInitCaller
	size    int
	times   uint64
}

func NewFixedPool(size int, factory CallerFactory) *PoolSender {
	cp := &PoolSender{
		size:    size,
		callers: make([]*LateInitCaller, size),
	}
	for i := 0; i < size; i++ {
		cp.callers[i] = &LateInitCaller{factory: factory}
	}
	return cp
}

func (c *PoolSender) Send(method string, ctx context.Context, v []interface{}, resp interface{}) error {
	delay := c.callers[atomic.AddUint64(&c.times, 1)%uint64(c.size)]
	client, err := delay.Get(ctx)
	if err != nil {
		return err
	}
	err = client.Caller.Call( method, v, resp)
	if err != nil {
		if err == ErrShutdown {
			delay.Clear(client.Version)
		}
	}
	return err
}
