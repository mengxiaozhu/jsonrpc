package jsonrpc

import (
	"context"
	"net"
)

type Addr func() (string, error)

type ConnGetter func(ctx context.Context) (net.Conn, error)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
