package rpc

import (
	"errors"
)

var (
	ErrEmptyParam   = errors.New("rpc: param list is empty")
	ErrLenMismatch  = errors.New("rpc: parameters length mismatch")
	ErrTypeMismatch = errors.New("rpc: type mismatch")
)

func lenHelper(args []interface{}) error {
	if len(args) == 0 {
		return ErrEmptyParam
	}

	return nil
}

func Uint16Param(u uint16) Param {
	return ParamFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(*uint16); ok {
			*p = u
			return nil
		}

		return ErrTypeMismatch
	})
}

func Uint32Param(u uint32) Param {
	return ParamFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(*uint32); ok {
			*p = u
			return nil
		}

		return ErrTypeMismatch
	})
}

func StringParam(s string) Param {
	return ParamFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(*string); ok {
			*p = s
			return nil
		}

		return ErrTypeMismatch
	})
}

func ByteSliceParam(b []byte) Param {
	return ParamFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(*[]byte); ok {
			*p = b
			return nil
		}

		return ErrTypeMismatch
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

func StringResult(p *string) Result {
	return ResultFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if s, ok := args[0].(string); ok {
			*p = s
			return nil
		}

		return ErrTypeMismatch
	})
}

func ByteSliceResult(b *[]byte) Result {
	return ResultFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if slice, ok := args[0].([]byte); ok {
			*b = slice
			return nil
		}

		return ErrTypeMismatch
	})
}

func StringSliceResult(s *[]string) Result {
	return ResultFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if slice, ok := args[0].([]string); ok {
			*s = slice
			return nil
		}

		return ErrTypeMismatch
	})
}

func Uint32Result(u *uint32) Result {
	return ResultFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(uint32); ok {
			*u = p
			return nil
		}

		return ErrTypeMismatch
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
