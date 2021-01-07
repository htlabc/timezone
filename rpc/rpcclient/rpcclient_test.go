package rpcclient

import (
	"testing"
)

func Test_Rpcclient(t *testing.T) {

	//go func(){
	//	client := NewRpcClient()
	//	votereq := &request.VoteRequest{"s1", 1}
	//	jsdat, _ := json.Marshal(votereq)
	//	req := &request.Request{CMD: request.V_ELECTION, URL: "127.0.0.1:5001", OBJ: jsdat}
	//	response := client.Send(req)
	//	resutl := &request.VoteResponse{}
	//	fmt.Println(string(response.Data.([](byte))))
	//	json.Unmarshal(response.Data.([]byte), resutl)
	//	fmt.Println(resutl)
	//}()

	//s:=server.GETINSTANCE()

	//p1 := new(server.Peer).Address("127.0.0.1:5001")
	//peers := make([]server.Peer, 0)
	//peers = append(peers, *p1)
	//s.SetPeerSet(peers, p1)
	//s.ElectionScheduler(time.NewTicker(3*time.Second))

}
