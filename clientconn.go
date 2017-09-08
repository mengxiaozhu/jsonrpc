package jsonrpc

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type callback chan responseAndError

type callbacks struct {
	store map[uint64]callback
	mutex sync.Mutex
}

func (c *callbacks) Add(num uint64) callback {
	cb := make(callback)
	c.mutex.Lock()
	c.store[num] = cb
	c.mutex.Unlock()
	return cb
}
func (c *callbacks) ReleaseAll(err error) {
	c.mutex.Lock()
	for _, cb := range c.store {
		cb <- responseAndError{error: err}
	}
	c.store = map[uint64]callback{}
	c.mutex.Unlock()
}
func (c *callbacks) Del(num uint64) {
	c.mutex.Lock()
	delete(c.store, num)
	c.mutex.Unlock()
}
func (c *callbacks) Init() {
	c.store = map[uint64]callback{}
}
func (c *callbacks) Notify(response *response) {
	c.mutex.Lock()
	if cb, ok := c.store[response.ID]; ok {
		delete(c.store, response.ID)
		cb <- responseAndError{response: response}
	}
	c.mutex.Unlock()
}

const (
	ClientClosed = 1
)

type ClientConn struct {
	request      request
	writerLocker sync.Mutex
	closed       int64
	conn         io.ReadWriteCloser
	decoder      *json.Decoder
	encoder      *json.Encoder
	sequence     uint64
	callbacks    callbacks
}

type ClientConnCallerFactory struct {
	target string
}

func (c *ClientConnCallerFactory) Create(ctx context.Context) (Caller, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.target)
	if err != nil {
		return nil, err
	}
	return NewClientConn(conn), nil
}

func NewClientConn(conn io.ReadWriteCloser) *ClientConn {
	c := &ClientConn{
		conn: conn,
		request: request{
			Version: "2.0",
		},
		decoder:  json.NewDecoder(conn),
		encoder:  json.NewEncoder(conn),
		sequence: 0,
	}
	c.callbacks.Init()
	go c.receiveResponse()
	return c
}

type request struct {
	Version string        `json:"jsonrpc"`
	Params  []interface{} `json:"params"`
	Method  string        `json:"method"`
	ID      uint64        `json:"id"`
}
type responseError struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func (r *responseError) Error() string {
	return r.Message
}

type responseAndError struct {
	*response
	error
}
type response struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *responseError  `json:"error"`
	ID      uint64          `json:"id"`
}

func (c *ClientConn) receiveResponse() {
	for {
		resp := &response{}
		err := c.decoder.Decode(resp)
		if err != nil {
			atomic.StoreInt64(&c.closed, ClientClosed)
			if _, ok := err.(*net.OpError); err == io.EOF || ok {
				err = ErrShutdown
			}
			c.callbacks.ReleaseAll(err)
			c.conn.Close()
			return
		}
		c.callbacks.Notify(resp)
	}
}
func (c *ClientConn) WriteRequest(serviceMethod string, args []interface{}) (cb callback, err error) {
	if atomic.LoadInt64(&c.closed) == ClientClosed {
		return nil, ErrShutdown
	}
	c.writerLocker.Lock()
	defer c.writerLocker.Unlock()
	if atomic.LoadInt64(&c.closed) == ClientClosed {
		return nil, ErrShutdown
	}
	c.request.ID = atomic.AddUint64(&c.sequence, 1)
	cb = c.callbacks.Add(c.request.ID)
	c.request.Params = args
	c.request.Method = serviceMethod
	err = c.encoder.Encode(c.request)
	if err != nil {
		c.callbacks.Del(c.request.ID)
		if _, ok := err.(*net.OpError); err == io.EOF || ok {
			atomic.StoreInt64(&c.closed, ClientClosed)
		}
	}
	return
}
func (c *ClientConn) Call(serviceMethod string, args []interface{}, reply interface{}) (err error) {
	if atomic.LoadInt64(&c.closed) == ClientClosed {
		return ErrShutdown
	}
	cb, err := c.WriteRequest(serviceMethod, args)
	if err != nil {
		return err
	}
	re := <-cb
	if re.error != nil {
		return re.error
	}
	if re.response.Error != nil {
		err = re.response.Error
		return
	}
	err = json.Unmarshal(re.response.Result, reply)
	return err
}
