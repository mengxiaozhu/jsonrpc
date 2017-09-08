package jsonrpc

import (
	"context"
	"net"
)

type Sender func(name string, ctx context.Context, input []interface{}, output interface{}) error

type Caller interface {
	Call(serviceMethod string, args []interface{}, reply interface{}) error
}

type CallerFactory func(ctx context.Context) (Caller, error)

type Addr func() (string, error)

type ConnGetter func(ctx context.Context) (net.Conn, error)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
