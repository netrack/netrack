package rpc

import (
	"errors"
)

var (
	ErrEmptyParam  = errors.New("rpc: param list is empty")
	ErrInvalidType = errors.New("rpc: invalid type")
)

func lenHelper(args []interface{}) error {
	if len(args) == 0 {
		return ErrEmptyParam
	}

	return nil
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

		return ErrInvalidType
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

		return ErrInvalidType
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

		return ErrInvalidType
	})
}

func ProcCallerResult(c *ProcCaller) Result {
	return ResultFunc(func(args ...interface{}) error {
		if err := lenHelper(args); err != nil {
			return err
		}

		if p, ok := args[0].(ProcCaller); ok {
			*c = p
			return nil
		}

		return ErrInvalidType
	})
}

//func CompositeParam(params ...Param) Param {
//return ParamFunc(func(args ...interface{}) error {
//for index := range params {
////param.Obtain(
//}
//})
//}
