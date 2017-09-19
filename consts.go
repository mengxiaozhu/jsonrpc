package jsonrpc

const Version = "2.0"

var (
	MethodNotFoundResponseError = &responseError{
		Code:    -32601,
		Message: "Method not found",
	}
	OverServerLimitError = &responseError{
		Code:    -32001,
		Message: "Over Server Limit",
	}
)
