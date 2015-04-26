package rpc

import (
	"errors"
	"reflect"
)

var (
	ErrEmptyParam   = errors.New("rpc: param list is empty")
	ErrLenMismatch  = errors.New("rpc: parameters length mismatch")
	ErrTypeMismatch = errors.New("rpc: type mismatch")
	ErrUnexpected   = errors.New("rpc: unexpected error")
)

func LenHelper(args []interface{}) error {
	if len(args) == 0 {
		return ErrEmptyParam
	}

	return nil
}

func setValue(first interface{}, second interface{}) (err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		err = ErrUnexpected

		if recoveredErr, ok := recovered.(error); ok {
			err = recoveredErr
		}
	}()

	firstValue := reflect.ValueOf(first)
	// Receiver should be a pointer and can be changed.
	if firstValue.Kind() != reflect.Ptr || !firstValue.Elem().CanSet() {
		return ErrTypeMismatch
	}

	secondValue := reflect.ValueOf(second)
	// Values shoud be the same type.
	if firstValue.Elem().Type() != secondValue.Type() {
		return ErrTypeMismatch
	}

	firstValue.Elem().Set(secondValue)
	return nil
}

func MakeParam(param interface{}) Param {
	return ParamFunc(func(args ...interface{}) (err error) {
		if err := LenHelper(args); err != nil {
			return err
		}

		return setValue(args[0], param)
	})
}

func CompositeParam(params ...Param) Param {
	return ParamFunc(func(args ...interface{}) error {
		if len(args) != len(params) {
			return ErrLenMismatch
		}

		for index, param := range params {
			if err := param.Obtain(args[index]); err != nil {
				return err
			}
		}

		return nil
	})
}

func MakeResult(result interface{}) Result {
	return ResultFunc(func(args ...interface{}) (err error) {
		if err := LenHelper(args); err != nil {
			return err
		}

		return setValue(result, args[0])
	})
}

func CompositeResult(results ...Result) Result {
	return ResultFunc(func(args ...interface{}) error {
		if len(args) != len(results) {
			return ErrLenMismatch
		}

		for index, result := range results {
			if err := result.Return(args[index]); err != nil {
				return err
			}
		}

		return nil
	})
}
