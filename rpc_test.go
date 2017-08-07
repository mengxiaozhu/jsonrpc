package jsonrpc

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"testing"
	"time"
)

const serviceName = "halo"

type ServerInterface struct {
	Add func(i int) (j int, err error) `rpc:"add"`
}

func TestRPC(t *testing.T) {
	go server()
	dialer := &net.Dialer{}
	f := &Factory{
		Context: context.Background(),
		Sender: NewFixedPool(20, func(ctx context.Context) (Caller, error) {
			conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:12345")
			if err != nil {
				return nil, err
			}
			return jsonrpc.NewClient(conn), nil

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
		fmt.Println(itfc.Add(90))
	}
}

type Impl struct {
}

func (f *Impl) Add(i int, j *int) (err error) {
	*j = i / 2
	return nil
}
func server() {
	listener, _ := net.Listen("tcp", ":12345")
	server := rpc.NewServer()
	server.RegisterName(serviceName, &Impl{})
	conn, err := listener.Accept()
	if err != nil {
		panic(err)
	}
	go server.ServeCodec(jsonrpc.NewServerCodec(conn))
	time.AfterFunc(time.Second, func() {
		conn.Close()
	})
	listener.Close()
	return
}
