package client

import (
	"errors"
	"log"
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
