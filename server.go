package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
)

var DefaultFilter = func(s string) string {
	return s
}

var DefaultServer = NewServer()

func NewServer() *Server {
	return &Server{
		functions: map[string]reflect.Value{},
		filter:    DefaultFilter,
	}
}

func Listen(laddr string) error {
	return DefaultServer.Run(laddr)
}

type Server struct {
	functions map[string]reflect.Value
	filter    func(string) string
}

func (server *Server) Run(laddr string) error {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go (&serverConn{
			server:          server,
			ReadWriteCloser: conn,
			Decoder:         json.NewDecoder(conn),
			Encoder:         json.NewEncoder(conn),
		}).Read()
	}
}
func (server *Server) Register(name string, obj interface{}) {
	value := reflect.ValueOf(obj)
	num := value.NumMethod()
	for i := 0; i < num; i++ {
		method := value.Type().Method(i)
		if method.Type.NumOut() == 2 && method.Type.Out(1) == emptyErrorType {
			server.functions[name+"."+server.filter(method.Name)] = value.Method(i)
		}
	}
}

func Register(name string, obj interface{}) {
	DefaultServer.Register(name, obj)
}

func ServeConn(conn io.ReadWriteCloser) {
	(&serverConn{
		server:          DefaultServer,
		ReadWriteCloser: conn,
		Decoder:         json.NewDecoder(conn),
		Encoder:         json.NewEncoder(conn),
	}).Read()
}

type ServerRequest struct {
	Version string            `json:"jsonrpc"`
	Params  []json.RawMessage `json:"params"`
	Method  string            `json:"method"`
	ID      uint64            `json:"id"`
}

type serverResponse struct {
	Version string         `json:"jsonrpc"`
	Result  interface{}    `json:"result"`
	Error   *responseError `json:"error"`
	ID      uint64         `json:"id"`
}

type serverConn struct {
	io.ReadWriteCloser
	*json.Encoder
	*json.Decoder
	mutex  sync.Mutex
	server *Server
}

func (c *serverConn) Read() {
	for {
		req := &ServerRequest{}
		err := c.Decode(req)
		if err != nil {
			c.Close()
			return
		}
		go c.server.serve(req, c)
	}
}
func (c *serverConn) Write(s *serverResponse) {
	c.mutex.Lock()
	err := c.Encode(s)
	c.mutex.Unlock()
	if err != nil {
		c.Close()
	}
}
func catchMethodPanic(writer *serverConn, ID uint64) {
	err := recover()
	if err != nil {
		writer.Write(&serverResponse{
			ID: ID,
			Error: &responseError{
				Code:    50000,
				Message: fmt.Sprint(err),
			},
		})
	}
}
func (server *Server) serve(request *ServerRequest, writer *serverConn) {
	defer catchMethodPanic(writer, request.ID)
	fn, has := server.functions[request.Method]
	if !has {
		writer.Write(&serverResponse{
			ID: request.ID,
			Error: &responseError{
				Code:    -32601,
				Message: "Method not found",
			},
		})
		return
	}
	inNum := fn.Type().NumIn()
	args := []reflect.Value{}

	for i := 0; i < inNum; i++ {
		arg := reflect.New(fn.Type().In(i))
		json.Unmarshal(request.Params[i], arg.Interface())
		args = append(args, arg.Elem())
	}
	resp := fn.Call(args)
	if resp[1].IsNil() {
		writer.Write(&serverResponse{
			ID:     request.ID,
			Result: resp[0].Interface(),
		})
	} else {
		writer.Write(&serverResponse{
			ID: request.ID,
			Error: &responseError{
				Code:    50000,
				Message: fmt.Sprint(resp[1].Interface()),
			},
		})
	}
}
