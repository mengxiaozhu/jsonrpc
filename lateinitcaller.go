package jsonrpc

import (
	"context"
	"sync"
	"sync/atomic"
)

type LateInitCaller struct {
	factory CallerFactory
	client  atomic.Value
	lock    sync.Mutex
	version uint64
}

func NewLateInitCaller(factory CallerFactory) *LateInitCaller {
	return &LateInitCaller{
		factory: factory,
	}
}

func (c *LateInitCaller) Get(ctx context.Context) (VersionedCaller, error) {
	client := c.client.Load()
	if client == nil || client == emptyVersionedCaller {
		c.lock.Lock()
		client = c.client.Load()
		if client != nil && client != emptyVersionedCaller {
			c.lock.Unlock()
			return client.(VersionedCaller), nil
		}
		caller, err := c.factory(ctx)
		if err != nil {
			c.lock.Unlock()
			return emptyVersionedCaller, err
		}
		nClient := VersionedCaller{Caller: caller, Version: atomic.LoadUint64(&c.version)}
		c.client.Store(nClient)
		c.lock.Unlock()
		return nClient, nil
	} else {
		return client.(VersionedCaller), nil
	}
}
func (c *LateInitCaller) Clear(version uint64) {
	ok := atomic.CompareAndSwapUint64(&c.version, version, version+1)
	if !ok {
		return
	}
	c.lock.Lock()
	c.client.Store(emptyVersionedCaller)
	c.lock.Unlock()
}
