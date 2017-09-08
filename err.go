package jsonrpc

import "errors"

var (
	ErrShutdown                            = errors.New("connection may be shutdown")
	ErrorInjectObjectMustBePointerOfStruct = errors.New("inject object must be pointer of struct")
)
