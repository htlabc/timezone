package server

import (
	"encoding/json"
	"fmt"
	"htl.com/request"
	"htl.com/rpc/rpcclient"
	"testing"
	"time"
)

func Test_Server(T *testing.T) {

	go func() {
		client := rpcclient.NewRpcClient()
		votereq := &request.VoteRequest{"s1", 1}
		jsdat, _ := json.Marshal(votereq)
		req := &request.Request{CMD: request.V_ELECTION, URL: "127.0.0.1:5001", OBJ: jsdat}
		response := client.Send(req)
		resutl := &request.VoteResponse{}
		fmt.Println(string(response.Data.([](byte))))
		json.Unmarshal(response.Data.([]byte), resutl)
		fmt.Println(resutl)
	}()

	s := GETINSTANCE()
	p1 := new(Peer).Address("127.0.0.1:5001")
	//p2 := new(server.Peer).Address("127.0.0.1:8002")
	//p3 := new(server.Peer).Address("127.0.0.1:8003")
	peers := make([]Peer, 0)
	peers = append(peers, *p1)
	s.SetPeerSet(peers, p1)
	s.ElectionScheduler(time.NewTicker(3 * time.Second))

}
