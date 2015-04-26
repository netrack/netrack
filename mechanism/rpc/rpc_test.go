package rpc

import (
	"testing"
)

func TestProcCaller(t *testing.T) {
	caller := New()

	var result int
	err := caller.Call(Type("function"), MakeParam(1), MakeResult(&result))
	if err == nil {
		t.Fatal("Failed to report error on unregistered function call")
	}

	squareFunc := func(param Param, result Result) error {
		var value int

		err = param.Obtain(&value)
		if err != nil {
			t.Fatal("Failed to obtain function paramaters:", err)
		}

		err = result.Return(value * value)
		if err != nil {
			t.Fatal("Failed to return result of function:", err)
		}

		return nil
	}

	caller.RegisterFunc(Type("function"), squareFunc)

	err = caller.Call(Type("function"), MakeParam(2), MakeResult(&result))
	if err != nil {
		t.Fatal("Failed to execute registered function:", err)
	}

	if result != 4 {
		t.Fatal("Failed to result right result of function:", result)
	}
}
