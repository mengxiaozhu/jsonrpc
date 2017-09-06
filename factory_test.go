package jsonrpc

import (
	"context"
	"errors"
	"testing"
	"time"
)

type structForTest struct {
	Add      func(name string) (i int, err error) `rpc:"add"`
	Error    func(name string) (i int, err error)
	NoMethod func(name string) (i int, err error)
}

func BenchmarkFactory_Inject(b *testing.B) {
	factory := Factory{
		Sender: func(name string, ctx context.Context, input []interface{}, output interface{}) error {
			switch name {
			case "serv.add":
				ptr := output.(*int)
				*ptr = 3 / 2
				return nil
			case "serv.Error":
				return errors.New("an error")
			}
			return errors.New("no such method")
		},
		Timeout: time.Minute,
		Context: context.Background(),
	}
	sft := &structForTest{}
	factory.Inject("serv", sft)
	b.Run("add", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sft.Add("")
		}
	})
	b.Run("pa", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				sft.Add("")
			}
		})
	})
}

func TestFactory_Inject(t *testing.T) {
	factory := Factory{
		Sender: func(name string, ctx context.Context, input []interface{}, output interface{}) error {
			switch name {
			case "serv.add":
				ptr := output.(*int)
				*ptr = 3 / 2
				return nil
			case "serv.Error":
				return errors.New("an error")
			}
			return errors.New("no such method")
		},
		Timeout: time.Minute,
		Context: context.Background(),
	}
	sft := &structForTest{}
	factory.Inject("serv", sft)
	result, err := sft.Add("")
	if err != nil {
		t.Fail()
		return
	}
	if result != 3/2 {
		t.Fail()
		return
	}
	result, err = sft.Error("")
	if err == nil {
		t.Fail()
		return
	}
	if err.Error() != "an error" {
		t.Fail()
		return
	}
	result, err = sft.NoMethod("")
	if err == nil {
		t.Fail()
		return
	}
	if err.Error() != "no such method" {
		t.Fail()
		return
	}
}
