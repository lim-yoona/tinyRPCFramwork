package encode

import (
	"io"
)

// 考虑到客户端的一次RPC请求，其中包括服务名，方法名，参数
// 服务端返回返回值和error
// 将请求和响应中的参数和返回值抽象为body
// 而把error以及服务名方法名抽象为Header
type Header struct {
	ServiceMethod string
	Seq           uint64 //请求序号，用于区分不同的请求
	Error         string
}

// 抽象出对消息体编码的接口,因为可能会有多种编解码方式
type Encoder interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// 定义编码函数，函数接收一个接口
type NewEncodeFun func(closer io.ReadWriteCloser) Encoder

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// 定义一个map，可以通过不同的Type（编码方式）返回构造函数
var EncodeFunMap map[Type]NewEncodeFun

func init() {
	// 为map开个空间，否则不能往里存东西
	EncodeFunMap = make(map[Type]NewEncodeFun)
	// 为Gob方法存一个构造函数进来
	EncodeFunMap[GobType] = NewGobEncoder
}
