package jsonrpc

const Version = "2.0"
const (
	MethodNotFoundCode  responseErrorCode = -32601
	ReturnErrorCode     responseErrorCode = -32001
	PanicErrorCode      responseErrorCode = -32002
	OverServerLimitCode responseErrorCode = -32003
)

var (
	MethodNotFoundResponseError = &responseError{
		Code:    MethodNotFoundCode,
		Message: "Method not found",
	}
	OverServerLimitError = &responseError{
		Code:    OverServerLimitCode,
		Message: "Over Server Limit",
	}
)
