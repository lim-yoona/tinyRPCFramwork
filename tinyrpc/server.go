package tinyrpc

import (
	"log"
	"net"
	"tinyRPCFramwork/encode"
)

// rpc通信的协议的数据结构需要我们自己设计
// 需要协商消息的编码方式，放在结构体option中
const MagicNumber = 0x3bef5c

type Option struct {
	// 标识这是一个rpc请求
	MagicNumber int
	// 定义编码方式
	EncodeType encode.Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	EncodeType:  encode.GobType,
}

// 报文协商采用Json编码Option，Header和Body编码方式由Option中的EncodeType决定
// 服务端首先使用 JSON 解码 Option，然后通过 Option 的 CodeType 解码剩余的内容。

// 实现服务端
// 一个服务端
type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (server *Server) Accept(listen net.Listener) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("获取连接出错", err)
			return
		}
		// 开一个协程去处理当前的连接
		go server.ServeConn(conn)
	}
}

func Accept(listen net.Listener) { DefaultServer.Accept(listen) }

// 处理请求
func (server *Server) ServerConn(conn *net.Conn) {

}
