package diyrpc

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
	"tinyRPCFramwork/irpc"
	"tinyRPCFramwork/service"
)

const MarkDiyrpc = 0x3bef5c

type Option struct {
	// 标记这是一本rpc消息
	MarkedDiyrpc      int
	CodeType          irpc.Type
	ConnectionTimeout time.Duration
	HandleTimeout     time.Duration
}

var invalidRequest = struct{}{}

var DefaultOption = &Option{
	MarkedDiyrpc:      MarkDiyrpc,
	CodeType:          irpc.GobType,
	ConnectionTimeout: time.Second * 10,
}

type request struct {
	h           *irpc.Header
	argv, reply reflect.Value
	mType       *service.MethodType
	svc         *service.Service
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

	req.svc, req.mType, err = s.findService(h.ServiceMethod)
	if err != nil {
		return nil, err
	}
	//req.argv = reflect.New(reflect.TypeOf(" "))
	req.argv = req.mType.NewArgv()
	req.reply = req.mType.NewReply()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err := code.ReadBody(argvi); err != nil {
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
	defer wg.Done()
	err := req.svc.Call(req.mType, req.argv, req.reply)
	if err != nil {
		req.h.Error = err.Error()
		s.sendResponse(code, req.h, invalidRequest, sending)
	}
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
	if _, dup := s.serviceMap.LoadOrStore(service.Name, service); dup {
		return errors.New("[rpc server] service already defined:" + service.Name)
	}
	return nil
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func (s *Server) findService(serviceMethod string) (sev *service.Service, mType *service.MethodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("[rpc server] serviceMethod 格式错误" + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svc, ok := s.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("[rpc server] can't find service" + serviceName)
	}
	sev = svc.(*service.Service)
	mType = sev.Method[methodName]
	return
}
