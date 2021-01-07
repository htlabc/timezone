package main

import (
	"fmt"
	"htl.com/rpc/rpcserver"
	"htl.com/server"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	Main()
}

func Main() {
	var stopch chan struct{} = make(chan struct{})
	s1 := server.NewServer()
	s2 := server.NewServer()
	s3 := server.NewServer()
	p1 := new(server.Peer).Address("127.0.0.1:8001")
	p2 := new(server.Peer).Address("127.0.0.1:8002")
	p3 := new(server.Peer).Address("127.0.0.1:8003")
	peers := make([]server.Peer, 0)
	peers = append(peers, *p1, *p2, *p3)
	s1.SetPeerSet(peers, p1)
	s2.SetPeerSet(peers, p2)
	s3.SetPeerSet(peers, p3)
	server := rpcserver.NewRpcServer()
	server1 := rpcserver.NewRpcServer()
	server2 := rpcserver.NewRpcServer()
	//server.Service = s
	rpcserver.ServerQueue[s1.GetSelfPeer().GetAddress()] = s1
	rpcserver.ServerQueue[s2.GetSelfPeer().GetAddress()] = s2
	rpcserver.ServerQueue[s3.GetSelfPeer().GetAddress()] = s3

	go rpcserver.Start(server, p1.GetAddress(), stopch)
	go s1.Start(stopch)

	go rpcserver.Start(server1, p2.GetAddress(), stopch)
	go s2.Start(stopch)

	go rpcserver.Start(server2, p3.GetAddress(), stopch)
	go s3.Start(stopch)

	time.Sleep(30 * time.Second)
	fmt.Println(s1.GetTimeZone())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
}
