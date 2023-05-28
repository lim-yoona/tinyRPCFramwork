package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
	"tinyRPCFramwork/code"
	"tinyRPCFramwork/diyrpc"
	"tinyRPCFramwork/irpc"
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
	addr := make(chan string)
	go startServe(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	_ = json.NewEncoder(conn).Encode(diyrpc.DefaultOption)
	codec := code.NewGobCode(conn)
	for i := 0; i < 10; i++ {
		h := &irpc.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		codec.Write(h, fmt.Sprintf("rpc req %d", h.Seq))
		codec.ReadHeader(h)
		var reply string
		codec.ReadBody(&reply)
		fmt.Println("reply:", reply)
	}
}
