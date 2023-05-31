package main

import (
	"log"
	"net"
	"sync"
	"time"
	client2 "tinyRPCFramwork/client"
	"tinyRPCFramwork/code"
	"tinyRPCFramwork/diyrpc"
)

// demo3.go对增加了服务注册功能的rpc框架进行测试
type Foo int
type Args struct {
	Num1 int
	Num2 int
}

func (f *Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}
func startServer(addr chan string) {
	var foo Foo
	if err := diyrpc.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	l, err := net.Listen("tcp", "127.0.0.1:19990")
	if err != nil {
		log.Println("[main] net.Listen err:", err)
	}
	log.Println("[main] Serve on :", l.Addr())
	addr <- l.Addr().String()
	diyrpc.Accept(l)
}

func main() {
	code.Init()
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	client, _ := client2.Dial("tcp", <-addr)
	defer func() {
		client.Close()
	}()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(int2 int) {
			defer wg.Done()
			args := &Args{
				Num1: int2 * 10,
				Num2: int2 * 20,
			}
			reply := new(int)
			if err := client.Call("Foo.Sum", args, reply); err != nil {
				log.Fatal("[main] call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, *reply)
		}(i)
	}
	wg.Wait()
}
