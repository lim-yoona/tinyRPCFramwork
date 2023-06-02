package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
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

type clientResult struct {
	client *Client
	err    error
}
type newClientFunc func(conn net.Conn, opt *diyrpc.Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*diyrpc.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address, opt.ConnectionTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{
			client: client,
			err:    err,
		}
	}()
	if opt.ConnectionTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	case <-time.After(opt.ConnectionTimeout):
		return nil, fmt.Errorf("[rpc client] connect create timeout:expect within %s", opt.ConnectionTimeout)
	case result := <-ch:
		return result.client, result.err
	}
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
		err := fmt.Errorf("[Client] Invaild CodeType: ", opt.CodeType)
		//log.Println(opt.CodeType)
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

// 设置opts为可选参数
func parseOptions(opts ...*diyrpc.Option) (*diyrpc.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return diyrpc.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("The number of options is more than 1")
	}
	opt := opts[0]
	opt.MarkedDiyrpc = diyrpc.DefaultOption.MarkedDiyrpc
	if opt.CodeType == "" {
		opt.CodeType = diyrpc.DefaultOption.CodeType
	}
	// for test
	opt.CodeType = irpc.GobType
	return opt, nil
}
func Dial(network, address string, opts ...*diyrpc.Option) (client *Client, err error) {
	//opt, err := parseOptions(opts...)
	//if err != nil {
	//	log.Println("[Client] parseOptions err:", err)
	//	return nil, err
	//}
	//conn, err := net.Dial(network, address)
	//if err != nil {
	//	log.Println("[Client] net.Dial err:", err)
	//	return nil, err
	//}
	//defer func() {
	//	if client == nil {
	//		conn.Close()
	//	}
	//}()
	//return newClient(conn, opt)

	return dialTimeout(newClient, network, address, opts...)
}
func (c *Client) send(call *Call) {
	c.sending.Lock()
	defer c.sending.Unlock()

	seq, err := c.registerCall(call)
	if err != nil {
		log.Println("[Client] registerCall err:", err)
		call.Error = err
		call.done()
		return
	}
	c.header.ServiceMethod = call.ServiceMethod
	c.header.Seq = seq
	c.header.Error = ""
	if err := c.cc.Write(&c.header, call.Args); err != nil {
		call := c.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}
func (c *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("[Client] done channel unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		reply:         reply,
		Done:          done,
	}
	c.send(call)
	return call
}
func (c *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := <-c.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	select {
	case <-ctx.Done():
		c.removeCall(call.Seq)
		return errors.New("[rpc client] call failed")
	case call := <-call.Done:
		return call.Error
	}
}
