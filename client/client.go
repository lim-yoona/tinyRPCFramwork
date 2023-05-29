package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"tinyRPCFramwork/diyrpc"
	"tinyRPCFramwork/irpc"
)

// 封装一个结构体Call来承载一次RPC调用所要用到的信息
type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	reply         interface{}
	Error         error
	// Done是为异步设计的，告诉已经调用完成
	Done chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

type Client struct {
	cc       irpc.ICode
	opt      *diyrpc.Option
	sending  sync.Mutex
	header   irpc.Header
	mu       sync.Mutex
	seq      uint64
	pending  map[uint64]*Call
	closing  bool
	shutdown bool
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		return errors.New("[Client] The Client has closing...")
	}
	c.closing = true
	return c.cc.Close()
}

func (c *Client) IsAvailable() bool {
	return !c.shutdown && !c.closing
}
func (c *Client) registerCall(call *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.shutdown {
		log.Println("[Client] RegisterCall but client closing or shutdown")
		return 0, errors.New("[Client] The Client has closing...")
	}
	call.Seq = c.seq
	c.pending[call.Seq] = call
	c.seq++
	return call.Seq, nil
}
func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing || c.shutdown {
		log.Println("[Client] RegisterCall but client closing or shutdown")
		return nil
	}
	call, ok := c.pending[seq]
	if !ok {
		log.Println("[Client] seq Call not exist")
	}
	delete(c.pending, seq)
	return call
}

func (c *Client) terminateCalls(err error) {
	c.sending.Lock()
	defer c.sending.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.shutdown = true
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}
func (c *Client) receive() {
	var err error
	for err == nil {
		var h irpc.Header
		err = c.cc.ReadHeader(&h)
		if err != nil {
			log.Println("[Client] Read Header faild")
			break
		}
		// 表示这个call已经处理完，可以删除了
		call := c.removeCall(h.Seq)
		// 判断得到的call
		// 如果是空，表明不存在这个call，直接返回错误
		// 如果读到的头中包含错误信息，表明没有正确处理
		// 生成错误，告诉调用方调用结束
		switch {
		case call == nil:
			// 给一个nil读body，自然会返回一个err
			err = c.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = c.cc.ReadBody(nil)
			call.done()
		default:
			err = c.cc.ReadBody(call.reply)
			if err != nil {
				call.Error = errors.New("[Client] Reading body" + err.Error())
			}
			call.done()
		}
	}
	c.terminateCalls(err)
}
func newClientCode(cc irpc.ICode, opt *diyrpc.Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}
func newClient(conn net.Conn, opt *diyrpc.Option) (*Client, error) {
	f := irpc.NewCodeFuncMap[opt.CodeType]
	if f == nil {
		err := fmt.Errorf("[Client] Invaild CodeType", opt.CodeType)
		log.Println("[Client] NewCodeFuncMap err:", err)
		conn.Close()
		return nil, err
	}
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("[Client] Send Option faild: ", err)
		conn.Close()
		return nil, err
	}
	return newClientCode(f(conn), opt), nil
}
