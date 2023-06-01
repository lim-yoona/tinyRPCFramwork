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
	// main中传过来的rcvr是&foo
	// 但是程序在运行时并不知道rcvr是什么
	// 因为是一个空接口，可能接收到任何值
	// 因此需要反射得到它的值和类型还有它的方法
	s := new(Service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.Name = reflect.Indirect(s.rcvr).Type().Name()
	s.typ = reflect.TypeOf(rcvr)
	// 判断rcvr的名字是否是公开的（首字母大写）
	if !ast.IsExported(s.Name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.Name)
	}
	s.registerMethods()
	return s
}
func (s *Service) registerMethods() {
	// 将接收到的rcvr的方法取出来保存
	s.Method = make(map[string]*MethodType)
	// s.typ.NumMethod()获取类型的方法数量
	for i := 0; i < s.typ.NumMethod(); i++ {
		// 获取第i个方法的信息
		method := s.typ.Method(i)
		mType := method.Type
		// 判断方法的入参数量和出参数量是否符合rpc调用方法
		// 如果不符合，就跳过
		// 过滤掉不符合条件的方法
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		// 判断方法的返回值是否是error类型
		// 如果不是，则过滤掉
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		// 获取函数类型mType的第2个和第3个参数的类型
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuildinType(argType) || !isExportedOrBuildinType(replyType) {
			continue
		}
		// 注册方法
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
	// 获取到方法的函数名
	f := mt.Method.Func
	// 调用函数f
	returnV := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnV[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
