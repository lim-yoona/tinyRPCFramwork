package main

import (
	"fmt"
	"log"
	"net"
	"tinyRPCFramwork/diyrpc"
)

func startServe(addr chan string) {
	l, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Println("[main startServe]: Listen err:", err)
		return
	}
	fmt.Printf("rpc server started on:", l.Addr())
	addr <- l.Addr().String()
	diyrpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServe(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

}
