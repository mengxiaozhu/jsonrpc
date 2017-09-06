package jsonrpc

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sync"
)

var DefaultFilter = func(s string) string {
	return s
}

func NewTcpServer() *tcpServer {
	return &tcpServer{
		functions: map[string]reflect.Value{},
		filter:    DefaultFilter,
	}
}

type tcpServer struct {
	functions map[string]reflect.Value
	filter    func(string) string
}

func (t *tcpServer) Run(laddr string) error {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go (&tcpServerConn{
			t:       t,
			Conn:    conn,
			Decoder: json.NewDecoder(conn),
			Encoder: json.NewEncoder(conn),
		}).Read()
	}
}
func (t *tcpServer) Register(name string, obj interface{}) {
	value := reflect.ValueOf(obj)
	num := value.NumMethod()
	for i := 0; i < num; i++ {
		method := value.Type().Method(i)
		if method.Type.NumOut() == 2 && method.Type.Out(1) == emptyErrorType {
			t.functions[name+"."+t.filter(method.Name)] = value.Method(i)
		}
	}
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
type tcpServerConn struct {
	net.Conn
	*json.Encoder
	*json.Decoder
	mutex sync.Mutex
	t     *tcpServer
}

func (c *tcpServerConn) Read() {
	for {
		req := &ServerRequest{}
		err := c.Decode(req)
		if err != nil {
			c.Close()
			return
		}
		go c.t.serve(req, c)
	}
}
func (c *tcpServerConn) Write(s serverResponse) {
	c.mutex.Lock()
	err := c.Encode(s)
	c.mutex.Unlock()
	if err != nil {
		c.Close()
	}
}

func (t *tcpServer) serve(request *ServerRequest, writer *tcpServerConn) {
	defer func() {
		err := recover()
		if err != nil {
			writer.Write(serverResponse{
				ID: request.ID,
				Error: &responseError{
					Code:    50000,
					Message: fmt.Sprint(err),
				},
			})
		}
	}()
	fn, has := t.functions[request.Method]
	if !has {
		writer.Write(serverResponse{
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
		writer.Write(serverResponse{
			ID:     request.ID,
			Result: resp[0].Interface(),
		})
	} else {
		writer.Write(serverResponse{
			ID: request.ID,
			Error: &responseError{
				Code:    50000,
				Message: fmt.Sprint(resp[1].Interface()),
			},
		})
	}
}
