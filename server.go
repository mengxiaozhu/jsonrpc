package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
)

// handler rpc request and write response, not async
type ServerHandler interface {
	Handle(request *ServerRequest, writer ResponseWriter)
}

// write response
type ResponseWriter interface {
	Write(i *ServerResponse)
}

func NewServer() *Server {
	table := NewFunctionTable()
	return &Server{
		Registry:      table,
		ServerHandler: &AsyncHandler{ServerHandler: table},
	}
}

type Server struct {
	Registry
	ServerHandler
}

func (server *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go server.ServeConn(conn)
	}
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	NewServerConnCtx(conn, server.ServerHandler).Read()
}

func (server *Server) Listen(tcpAddr string) error {
	l, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return err
	}
	return server.Serve(l)
}

type AsyncHandler struct {
	ServerHandler
}

func (h *AsyncHandler) Handle(req *ServerRequest, resp ResponseWriter) {
	go h.ServerHandler.Handle(req, resp)
}

type Executor interface {
	Execute(request *ServerRequest, writer ResponseWriter)
}

type FunctionExecutor reflect.Value

func (executor FunctionExecutor) Execute(request *ServerRequest, writer ResponseWriter) {
	fn := reflect.Value(executor)
	inNum := fn.Type().NumIn()
	args := []reflect.Value{}

	for i := 0; i < inNum; i++ {
		arg := reflect.New(fn.Type().In(i))
		json.Unmarshal(request.Params[i], arg.Interface())
		args = append(args, arg.Elem())
	}

	defer recoverCallPanic(writer, request.ID)
	resp := fn.Call(args)
	if resp[1].IsNil() {
		writer.Write(&ServerResponse{
			ID:     request.ID,
			Result: resp[0].Interface(),
		})
	} else {
		writer.Write(&ServerResponse{
			ID: request.ID,
			Error: &responseError{
				Code:    ReturnErrorCode,
				Message: fmt.Sprint(resp[1].Interface()),
			},
		})
	}
}

func recoverCallPanic(writer ResponseWriter, ID uint64) {
	panicThing := recover()
	if panicThing != nil {
		writer.Write(CreateErrorResponse(ID, &responseError{
			Code:    PanicErrorCode,
			Message: fmt.Sprint(panicThing),
		}))
	}
}
