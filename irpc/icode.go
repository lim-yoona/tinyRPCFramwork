package irpc

import (
	"io"
	"tinyRPCFramwork/code"
)

type ICode interface {
	io.Closer
	ReadHeader(header *code.Header) error
	ReadBody(interface{}) error
	Write(*code.Header, interface{}) error
}
type NewCodeFunc func(closer io.ReadWriteCloser) ICode
type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodeFuncMap map[Type]NewCodeFunc

func init() {
	NewCodeFuncMap := make(map[Type]NewCodeFunc)
	NewCodeFuncMap[GobType] = code.NewGobCode
}
