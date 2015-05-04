package injector

import (
	"testing"
)

func TestInjector(t *testing.T) {
	i := New()

	var value int
	if i.Obtain(&value) == nil {
		t.Fatal("Failed report error on unconsistent types")
	}

	i.Bind(new(interface{}), 32)

	var interfaceValue interface{}
	if i.Obtain(&interfaceValue) != nil {
		t.Fatal("Failed to obtain bound type")
	}

	value, ok := interfaceValue.(int)
	if !ok {
		t.Fatal("Failed to set bound value:", interfaceValue)
	}

	if value != 32 {
		t.Fatal("Wrong value returned:", value)
	}

	// This call should not panic
	i.Bind(32, 32)
}
