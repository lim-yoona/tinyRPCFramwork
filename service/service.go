package service

import (
	"reflect"
	"sync/atomic"
)

// 通过反射实现结构体与服务的映射关系
type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

func (mt *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&mt.numCalls)
}
