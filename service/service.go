package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

// 通过反射实现结构体与服务的映射关系
type MethodType struct {
	Method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

func (mt *MethodType) NumCalls() uint64 {
	return atomic.LoadUint64(&mt.numCalls)
}
func (mt *MethodType) NewArgv() reflect.Value {
	var argv reflect.Value
	if mt.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mt.ArgType.Elem())
	} else {
		argv = reflect.New(mt.ArgType).Elem()
	}
	return argv
}
func (mt *MethodType) NewReply() reflect.Value {
	replyv := reflect.New(mt.ReplyType.Elem())
	switch mt.ReplyType.Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mt.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mt.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type Service struct {
	Name   string
	typ    reflect.Type
	rcvr   reflect.Value
	Method map[string]*MethodType
}

func NewService(rcvr interface{}) *Service {
	s := new(Service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.Name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.Name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.Name)
	}
	s.registerMethods()
	return s
}
func (s *Service) registerMethods() {
	s.Method = make(map[string]*MethodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i)
		mType := method.Type
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuildinType(argType) || !isExportedOrBuildinType(replyType) {
			continue
		}
		s.Method[method.Name] = &MethodType{
			Method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("[rpc server] register %s.%s\n", s.Name, method.Name)
	}
}
func isExportedOrBuildinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func (s *Service) Call(mt *MethodType, argv, replyv reflect.Value) error {
	// 被调用，调用次数+1
	atomic.AddUint64(&mt.numCalls, 1)
	f := mt.Method.Func
	returnV := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnV[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
