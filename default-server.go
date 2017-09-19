package jsonrpc

import "io"

var DefaultNameMapper = func(s string) string {
	return s
}

var DefaultServer = NewServer()

func Listen(tcpAddr string) error {
	return DefaultServer.Listen(tcpAddr)
}

func Register(name string, obj interface{}) {
	DefaultServer.Register(name, obj)
}

func ServeConn(conn io.ReadWriteCloser) {
	DefaultServer.ServeConn(conn)
}
