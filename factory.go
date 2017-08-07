package jsonrpc

import (
	"context"
	"errors"
	"reflect"
	"time"
)

var (
	ErrorInjectObjectMustBePointerOfStruct = errors.New("inject object must be pointer of struct")
)

type Sender func(name string, ctx context.Context, input interface{}, output interface{}) error

type Factory struct {
	MethodNameFilter func(string) string
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
			if f.MethodNameFilter != nil {
				methodName = f.MethodNameFilter(field.Name)
			} else {
				methodName = field.Name
			}
		}
		if structValue.Field(i).CanSet() {
			if field.Type.Kind() == reflect.Func && f.funcType(field.Type) == OneArgWithErrorReturn {
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
	return reflect.MakeFunc(fn, func(args []reflect.Value) (results []reflect.Value) {
		ctx, _ := context.WithTimeout(f.Context, f.Timeout)
		returnValue := reflect.New(resultType)
		err := f.Sender(serviceName+"."+methodName, ctx, args[0].Interface(), returnValue.Interface())
		if err == nil {
			return []reflect.Value{returnValue.Elem(), reflect.New(emptyErrorType).Elem()}
		} else {
			return []reflect.Value{returnValue.Elem(), reflect.ValueOf(&err).Elem()}
		}
	})
}
