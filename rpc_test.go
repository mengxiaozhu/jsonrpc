package jsonrpc

import (
	"context"
	"fmt"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net"
	"net/rpc"
	"testing"
	"time"
)

const serviceName = "halo"

type ServerInterface struct {
	Add func(i int) (j int, err error) `rpc:"Add"`
}

func TestRPC(t *testing.T) {
	go onceServer()
	dialer := &net.Dialer{}
	f := &Factory{
		Context: context.Background(),
		Sender: NewFixedPool(20, func(ctx context.Context) (Caller, error) {
			conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:12345")
			if err != nil {
				return nil, err
			}
			return NewTcpClient(conn), nil

		}).Send,
		Timeout: time.Second,
	}
	itfc := &ServerInterface{}
	err := f.Inject(serviceName, itfc)
	if err != nil {
		fmt.Println(err)
		return
	}
	for i := 0; i < 50; i++ {
		time.Sleep(time.Millisecond * 100)
		fmt.Println(itfc.Add(90 + i))
	}
}

type Impl struct {
}

func (f *Impl) Add(i int) (j int, err error) {
	return i / 2, nil
}
func BenchmarkRPC(b *testing.B) {
	go server()
	dialer := &net.Dialer{}
	f := &Factory{
		Context: context.Background(),
		Sender: NewFixedPool(40, func(ctx context.Context) (Caller, error) {
			conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:12346")
			if err != nil {
				return nil, err
			}
			return NewTcpClient(conn), nil

		}).Send,
		Timeout: time.Second,
	}
	itfc := &ServerInterface{}
	err := f.Inject(serviceName, itfc)
	if err != nil {
		panic(err)
		return
	}
	b.Run("benchmark", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				itfc.Add(2)
			}
		})
	})

}
func server() {
	listener, err := net.Listen("tcp", ":12346")
	if err != nil {
		panic(err)
	}
	server := rpc.NewServer()
	server.RegisterName(serviceName, &Impl{})
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go server.ServeCodec(jsonrpc2.NewServerCodec(conn, nil))
	}
}
func onceServer() {
	server := NewTcpServer()
	server.Register(serviceName, &Impl{})
	server.Run(":12345")
	return
}
