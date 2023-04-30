package tinyrpc

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
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
		go server.ServerConn(conn)
	}
}

func Accept(listen net.Listener) { DefaultServer.Accept(listen) }

// 处理请求
func (server *Server) ServerConn(conn io.ReadWriteCloser) {
	// 最后关闭连接
	defer func() { _ = conn.Close() }()

	// 定义一个对象接收json包的解码结果
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc err: Option decode error", err)
		return
	}
	// 取出对应编码方式的结构体构造函数
	f := encode.EncodeFunMap[opt.EncodeType]
	if f == nil {
		log.Println("rpc err: invalid encode type", opt.EncodeType)
		return
	}
	server.serveEncode(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveEncode(encoder *encode.Encoder) {
	// 制定一个锁，确保发送一个完整的response
	sending := new(sync.Mutex)
	// 等待所有请求处理完
	wg := new(sync.WaitGroup)
	for {
		req, err := server.readRequest(encoder)
		if err != nil {
			if req == nil {
				break
			}
			req.H.Error = err.Error()
			server.sendRequest(encoder, req.H, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(encoder, req, sending, wg)
	}
	wg.Wait()
	_ = encoder.Close()
}

type request struct {
	h            *encode.Header
	argv, replyv reflect.Value
}
