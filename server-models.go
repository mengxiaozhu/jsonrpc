package jsonrpc

import "encoding/json"

type ServerRequest struct {
	Version string            `json:"jsonrpc"`
	Params  []json.RawMessage `json:"params"`
	Method  string            `json:"method"`
	ID      uint64            `json:"id"`
}

type ServerResponse struct {
	Version string         `json:"jsonrpc"`
	Result  interface{}    `json:"result"`
	Error   *responseError `json:"error"`
	ID      uint64         `json:"id"`
}

func CreateErrorResponse(id uint64, err *responseError) *ServerResponse {
	return &ServerResponse{
		Version: Version,
		Error:   err,
		ID:      id,
	}
}
