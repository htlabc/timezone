package main

import (
	"fmt"
	"htl.com/server"
)

func main() {
	s := server.INSTANCE
	p1 := new(server.Peer).Address("127.0.0.1:8001")
	p2 := new(server.Peer).Address("127.0.0.1:8002")
	p3 := new(server.Peer).Address("127.0.0.1:8003")
	peers := make([]server.Peer, 0)
	peers = append(peers, *p2, *p3)
	s.SetPeerSet(peers, p1)
	s.Start()
	fmt.Println(s.GetTimeZone())
}
