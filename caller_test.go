package jsonrpc

import (
	"context"
	"github.com/uber-go/atomic"
	"sync"
	"testing"
)

type callerForTest struct {
}

func (c *callerForTest) Call(serviceMethod string, args []interface{}, reply interface{}) error {
	return nil
}
func TestLateInitCaller_Get(t *testing.T) {
	created := atomic.NewInt64(0)
	lateInitCaller := NewLateInitCaller(func(ctx context.Context) (Caller, error) {
		created.Add(1)
		return &callerForTest{}, nil
	})
	if created.Load() != 0 {
		t.Fail()
		return
	}
	ctx := context.Background()
	wg := sync.WaitGroup{}
	for i := 0; i < 3000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caller, err := lateInitCaller.Get(ctx)
			if err != nil {
				t.Fail()
				return
			}
			caller.Caller.Call("", nil, nil)
		}()
	}
	wg.Wait()
	if created.Load() != 1 {
		t.Log("fail:", created.Load())
		t.Fail()
		return
	}
}
func TestLateInitCaller_Clear(t *testing.T) {
	created := atomic.NewInt64(0)
	lateInitCaller := NewLateInitCaller(func(ctx context.Context) (Caller, error) {
		created.Add(1)
		return &callerForTest{}, nil
	})
	if created.Load() != 0 {
		t.Fail()
		return
	}
	ctx := context.Background()
	caller, err := lateInitCaller.Get(ctx)
	if err != nil {
		t.Fail()
	}
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lateInitCaller.Clear(caller.Version)
		}()
	}
	for i := 0; i < 3000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			caller, err := lateInitCaller.Get(ctx)
			if err != nil {
				t.Fail()
				return
			}
			caller.Caller.Call("", nil, nil)
		}()
	}
	wg.Wait()
	caller, err = lateInitCaller.Get(ctx)
	if err != nil {
		t.Fail()
		return
	}
	if caller.Version != 1 && created.Load() == 2 {
		t.Fail()
	}
	t.Log("success", caller.Version)

}
