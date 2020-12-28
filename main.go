package main

import (
	"htl.com/rpc/rpcserver"
	"htl.com/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	Main()
}

func Main() {
	var stopch chan struct{} = make(chan struct{})
	s := server.NewServer()
	p1 := new(server.Peer).Address("127.0.0.1:8001")
	//p2 := new(server.Peer).Address("127.0.0.1:8002")
	//p3 := new(server.Peer).Address("127.0.0.1:8003")
	peers := make([]server.Peer, 0)
	peers = append(peers, *p1)
	s.SetPeerSet(peers, p1)
	rpcserver := rpcserver.NewRpcServer()
	rpcserver.Service = s
	go rpcserver.Start(p1.GetAddress(), stopch)
	go s.Start(stopch)
	//fmt.Println(s.GetTimeZone())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
