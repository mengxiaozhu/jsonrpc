package jsonrpc

// limiter
type Limiter interface {
	Allow() bool
}

type LimitHandler struct {
	Limiter
	ServerHandler
}

func (l *LimitHandler) Handle(request *ServerRequest, writer ResponseWriter) {
	if l.Limiter.Allow() {
		l.ServerHandler.Handle(request, writer)
		return
	}
	writer.Write(CreateErrorResponse(request.ID, OverServerLimitError))
}

type MethodsLimitHandler struct {
	Limiters map[string]Limiter
	ServerHandler
}

func (l *MethodsLimitHandler) Handle(request *ServerRequest, writer ResponseWriter) {

	if limiter, ok := l.Limiters[request.Method]; ok {
		if !limiter.Allow() {
			writer.Write(CreateErrorResponse(request.ID, OverServerLimitError))
			return
		}
	}

	l.ServerHandler.Handle(request, writer)
}
