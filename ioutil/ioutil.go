package ioutil

import (
	"io"
)

type ReaderFromFunc func(io.Reader) (int64, error)

func (fn ReaderFromFunc) ReadFrom(r io.Reader) (int64, error) {
	return fn(r)
}

type WriterToFunc func(io.Writer) (int64, error)

func (fn WriterToFunc) WriteTo(w io.Writer) (int64, error) {
	return fn(w)
}
