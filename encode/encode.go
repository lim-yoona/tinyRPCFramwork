package encode

import "io"

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
