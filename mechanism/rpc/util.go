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

func MakeParam(param interface{}) Param {
	return ParamFunc(func(args ...interface{}) (err error) {
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

		if err := LenHelper(args); err != nil {
			return err
		}

		newValue := reflect.ValueOf(args[0])
		// Receiver should be a pointer and can be changed.
		if newValue.Kind() != reflect.Ptr || !newValue.Elem().CanSet() {
			return ErrTypeMismatch
		}

		paramValue := reflect.ValueOf(param)
		// Values shoud be the same type.
		if paramValue.Type() != newValue.Elem().Type() {
			return ErrTypeMismatch
		}

		newValue.Elem().Set(paramValue)
		return nil
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

		if err := LenHelper(args); err != nil {
			return err
		}

		resultValue := reflect.ValueOf(result)
		// Result should be a pointer and can be changed.
		if resultValue.Kind() != reflect.Ptr || !resultValue.Elem().CanSet() {
			return ErrTypeMismatch
		}

		newValue := reflect.ValueOf(args[0])
		// Values shoud be the same type.
		if resultValue.Elem().Type() != newValue.Type() {
			return ErrTypeMismatch
		}

		resultValue.Elem().Set(newValue)
		return nil
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
