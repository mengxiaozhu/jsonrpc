package jsonrpc

import "reflect"

var _ Registry = &FunctionTable{}
var _ ServerHandler = &FunctionTable{}

type Registry interface {
	Register(name string, obj interface{})
	Find(method string) (fn Executor, has bool)
}

type FunctionTable struct {
	functions  map[string]FunctionExecutor
	nameMapper func(string) string
}

func (table *FunctionTable) Register(name string, obj interface{}) {
	value := reflect.ValueOf(obj)
	num := value.NumMethod()
	for i := 0; i < num; i++ {
		method := value.Type().Method(i)
		if method.Type.NumOut() == 2 && method.Type.Out(1) == emptyErrorType {
			table.functions[name+"."+table.nameMapper(method.Name)] = FunctionExecutor(value.Method(i))
		}
	}
}

func (table *FunctionTable) Find(method string) (fn Executor, has bool) {
	fn, has = table.functions[method]
	return
}

func (table *FunctionTable) Handle(req *ServerRequest, resp ResponseWriter) {
	fn, ok := table.Find(req.Method)
	if ok {
		fn.Execute(req, resp)
		return
	}

	resp.Write(&ServerResponse{
		Version: Version,
		Error:   MethodNotFoundResponseError,
	})
}
