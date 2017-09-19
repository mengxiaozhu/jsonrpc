package jsonrpc

import (
	"encoding/json"
	"io"
	"sync"
)

func NewServerConnCtx(conn io.ReadWriteCloser, handler ServerHandler) *serverConnCtx {
	return &serverConnCtx{
		handler:         handler,
		ReadWriteCloser: conn,
		Decoder:         json.NewDecoder(conn),
		Encoder:         json.NewEncoder(conn),
	}
}

type serverConnCtx struct {
	io.ReadWriteCloser
	*json.Encoder
	*json.Decoder
	mutex   sync.Mutex
	handler ServerHandler
}

func (c *serverConnCtx) Read() {
	for {
		req := &ServerRequest{}
		err := c.Decode(req)
		if err != nil {
			c.Close()
			return
		}
		// maybe block
		c.handler.Handle(req, c)
	}
}

func (c *serverConnCtx) Write(s *ServerResponse) {
	err := c.Encode(s)
	if err != nil {
		c.Close()
	}
}
