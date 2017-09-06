package jsonrpc

import (
	"context"
	"errors"
	"net"
	"reflect"
	"time"
)

var (
	ErrorInjectObjectMustBePointerOfStruct = errors.New("inject object must be pointer of struct")
)

type Sender func(name string, ctx context.Context, input []interface{}, output interface{}) error

func New(target string, poolsize int) *Factory {
	factory := &Factory{}
	factory.Sender = NewFixedPool(poolsize, func(ctx context.Context) (Caller, error) {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", target)
		if err != nil {
			return nil, err
		}
		return NewTcpClient(conn), nil

	}).Send
	factory.Context = context.Background()
	factory.Timeout = 20 * time.Second
	return factory
}

type Factory struct {
	MethodNameMapper func(string) string
	Sender           Sender
	Context          context.Context
	Timeout          time.Duration
}

func (f *Factory) Inject(name string, obj interface{}) error {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Ptr && objVal.Type().Elem().Kind() != reflect.Struct {
		return ErrorInjectObjectMustBePointerOfStruct
	}
	structType := objVal.Type().Elem()
	structValue := objVal.Elem()

	numField := structType.NumField()
	for i := 0; i < numField; i++ {
		field := structType.Field(i)
		methodName := field.Tag.Get("rpc")
		if methodName == "" {
			if f.MethodNameMapper != nil {
				methodName = f.MethodNameMapper(field.Name)
			} else {
				methodName = field.Name
			}
		}
		if structValue.Field(i).CanSet() {
			if field.Type.Kind() == reflect.Func {
				structValue.Field(i).Set(f.makeFunc(name, methodName, field.Type))
			}
		}
	}
	return nil
}

type ValidFuncType uint

const (
	Invalid ValidFuncType = iota
	OneArgWithErrorReturn
)

func (f *Factory) funcType(fn reflect.Type) ValidFuncType {
	if fn.NumIn() != 1 || fn.NumOut() != 2 {
		return Invalid
	}
	if fn.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
		return Invalid
	}
	return OneArgWithErrorReturn
}

var emptyErr error
var emptyErrorType = reflect.TypeOf(&emptyErr).Elem()

func (f *Factory) makeFunc(serviceName string, methodName string, fn reflect.Type) reflect.Value {
	resultType := fn.Out(0)
	name := serviceName + "." + methodName
	fi := &methodInfo{
		name:       name,
		resultType: resultType,
		ctx:        f.Context,
		Timeout:    f.Timeout,
		Sender:     f.Sender,
	}
	return reflect.MakeFunc(fn, fi.Do)
}

type methodInfo struct {
	name       string
	resultType reflect.Type
	ctx        context.Context
	Sender     Sender
	Timeout    time.Duration
}

func (info *methodInfo) Do(args []reflect.Value) (results []reflect.Value) {
	ctx, _ := context.WithTimeout(info.ctx, info.Timeout)
	returnValue := reflect.New(info.resultType)
	params := []interface{}{}
	for _, v := range args {
		params = append(params, v.Interface())
	}
	err := info.Sender(info.name, ctx, params, returnValue.Interface())
	if err == nil {
		return []reflect.Value{returnValue.Elem(), reflect.New(emptyErrorType).Elem()}
	} else {
		return []reflect.Value{returnValue.Elem(), reflect.ValueOf(&err).Elem()}
	}
}
