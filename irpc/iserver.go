package irpc

import "net"

type IServer interface {
	Accept(listener net.Listener)
	Register(rcvr interface{}) error
}
