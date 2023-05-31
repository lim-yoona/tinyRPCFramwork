package diyrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"tinyRPCFramwork/irpc"
	"tinyRPCFramwork/service"
)

const MarkDiyrpc = 0x3bef5c

type Option struct {
	// 标记这是一本rpc消息
	MarkedDiyrpc int
	CodeType     irpc.Type
}

var invalidRequest = struct{}{}

var DefaultOption = &Option{
	MarkedDiyrpc: MarkDiyrpc,
	CodeType:     irpc.GobType,
}

type request struct {
	h           *irpc.Header
	argv, reply reflect.Value
}

// 采用json编码option，拿到option中的编码方式之后
// 用那种编码来编码body

type Server struct {
	serviceMap sync.Map
}

func NewServer() irpc.IServer {
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Accept(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.ServeConn(conn)
	}
}
func Accept(listener net.Listener) {
	DefaultServer.Accept(listener)
}
func (s *Server) ServeConn(conn net.Conn) {
	defer func() { conn.Close() }()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("json.NewDecoder(conn).Decode err", err)
		return
	}
	if opt.MarkedDiyrpc != MarkDiyrpc {
		log.Println("invalid MarkedDiyrpc")
		return
	}
	f := irpc.NewCodeFuncMap[opt.CodeType]
	if f == nil {
		log.Println("[rpc server]: invaild code type!", opt.CodeType)
		return
	}
	// f是对应编码方法类的构造函数
	s.serveCode(f(conn))
}

func (s *Server) serveCode(code irpc.ICode) {
	// Mutex make sure that serve return a complete response
	mu := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(code)
		if err != nil {
			if req == nil {
				break
			}
			// 如果读Header出错
			// 给客户端返回一个头中包含错误信息的消息
			req.h.Error = err.Error()
			s.sendResponse(code, req.h, invalidRequest, mu)
			continue
		}
		wg.Add(1)
		go s.handleRequest(code, req, mu, wg)
	}
	wg.Wait()
	code.Close()
}

func (s *Server) readRequest(code irpc.ICode) (*request, error) {
	h, err := s.readRequestHeader(code)
	if err != nil {
		return nil, err
	}
	req := &request{
		h: h,
	}
	// TODO:现在还不知道argv
	req.argv = reflect.New(reflect.TypeOf(" "))
	if err := code.ReadBody(req.argv.Interface()); err != nil {
		log.Println("[rpc server]: read argv faild:", err)
		return nil, err
	}
	return req, nil
}

func (s *Server) sendResponse(code irpc.ICode, h *irpc.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := code.Write(h, body); err != nil {
		log.Println("[rpc server]: write response err:", err)
	}
}
func (s *Server) handleRequest(code irpc.ICode, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO: 这块应该处理业务

	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.reply = reflect.ValueOf(fmt.Sprintf("rpc resp %d", req.h.Seq))
	s.sendResponse(code, req.h, req.reply.Interface(), sending)

}
func (s *Server) readRequestHeader(iCode irpc.ICode) (*irpc.Header, error) {
	var h irpc.Header
	if err := iCode.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("[rpc server]: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (s *Server) Register(rcvr interface{}) error {
	service := service.NewService(rcvr)
	if _, dup := s.serviceMap.LoadOrStore(service.Name, s); dup {

	}
}
