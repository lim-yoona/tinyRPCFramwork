package irpc

import (
	"io"
)

// message header
type Header struct {
	// service method name
	ServiceMethod string
	// message seq
	Seq uint64
	// 返回的错误信息
	Error string
}

type ICode interface {
	io.Closer
	ReadHeader(header *Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}
type NewCodeFunc func(closer io.ReadWriteCloser) ICode
type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodeFuncMap map[Type]NewCodeFunc
