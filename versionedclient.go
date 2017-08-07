package jsonrpc

type VersionedCaller struct {
	Caller  Caller
	Version uint64
}

var emptyVersionedCaller VersionedCaller
