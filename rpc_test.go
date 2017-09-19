package jsonrpc

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"testing"
	"time"
)

const serviceName = "halo"

type ServerInterface struct {
	Add func(i int) (j string, err error) `rpc:"add"`
}

func TestRPC(t *testing.T) {
	go onceServer()
	callerFactory := &ClientConnCallerFactory{
		target: "127.0.0.1:12345",
	}
	f := &Factory{
		Context: context.Background(),
		Sender:  NewFixedPool(20, callerFactory.Create).Send,
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

func (f *Impl) Add(i int) (j string, err error) {
	return strconv.Itoa(i / 2), nil
}

type Impl2 struct {
}

func (f *Impl2) Add(i int, j *string) (err error) {
	*j = strconv.Itoa(i / 2)
	return nil
}
func BenchmarkRPC3(b *testing.B) {
	go jsonrpcserver()
	time.Sleep(1 * time.Second)
	c, err := jsonrpc.Dial("tcp", "127.0.0.1:12346")
	if err != nil {
		panic(err)
	}
	b.Run("benchmark", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				k := ""
				c.Call(serviceName+"."+"Add", 2, &k)
			}
		})
	})
}
func BenchmarkRPC(b *testing.B) {
	go server()
	time.Sleep(1 * time.Second)

	callerFactory := &ClientConnCallerFactory{
		target: "127.0.0.1:12345",
	}
	factory := &Factory{
		Context: context.Background(),
		Sender:  NewFixedPool(5, callerFactory.Create).Send,
		Timeout: time.Second,
	}
	itfc := &ServerInterface{}
	err := factory.Inject(serviceName, itfc)
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

func BenchmarkRPC4(b *testing.B) {
	time.Sleep(1 * time.Second)

	callerFactory := &ClientConnCallerFactory{
		target: "127.0.0.1:12345",
	}
	factory := &Factory{
		Context: context.Background(),
		Sender:  NewFixedPool(10, callerFactory.Create).Send,
		Timeout: time.Second,
	}
	itfc := &ServerInterface{}
	err := factory.Inject(serviceName, itfc)
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
func BenchmarkRPC2(b *testing.B) {
	go jsonrpcserver()
	callerFactory := &ClientConnCallerFactory{
		target: "127.0.0.1:12346",
	}
	factory := &Factory{
		Context: context.Background(),
		Sender:  NewFixedPool(100, callerFactory.Create).Send,
		Timeout: time.Second,
	}
	itfc := &ServerInterface{}
	err := factory.Inject(serviceName, itfc)
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
func jsonrpcserver() {
	l, err := net.Listen("tcp", "127.0.0.1:12346")
	if err != nil {
		panic(err)
	}
	rpc.RegisterName(serviceName, &Impl2{})
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go jsonrpc.ServeConn(conn)
	}
}
func server() {
	server := NewServer()
	server.Register(serviceName, &Impl{})
	server.Listen(":12345")
	return
}
func onceServer() {
	server := NewServer()
	server.Register(serviceName, &Impl{})
	server.Listen(":12345")
	return
}
