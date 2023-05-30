package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	client2 "tinyRPCFramwork/client"
	"tinyRPCFramwork/code"
	"tinyRPCFramwork/diyrpc"
)

func startServe(addr chan string) {
	l, err := net.Listen("tcp", "127.0.0.1:19090")
	if err != nil {
		log.Println("[main startServe]: Listen err:", err)
		return
	}
	fmt.Println("rpc server started on:", l.Addr())
	addr <- l.Addr().String()
	diyrpc.Accept(l)
}

func main() {
	code.Init()
	log.SetFlags(0)

	addr := make(chan string)
	go startServe(addr)

	client, err := client2.Dial("tcp", <-addr)
	if err != nil {
		log.Println("main Dial err:", err)
		return
	}
	defer func() {
		client.Close()
	}()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(int2 int) {
			defer wg.Done()
			args := fmt.Sprintf("rpc req %d", i)
			var reply string
			if err := client.Call("Foo", args, &reply); err != nil {
				log.Println("[main] client Call err:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
