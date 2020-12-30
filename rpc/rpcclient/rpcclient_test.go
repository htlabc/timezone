package rpcclient

import (
	"encoding/json"
	"fmt"
	"htl.com/request"
	"testing"
)

func Test_Rpcclient(t *testing.T) {
	client := NewRpcClient()
	req := &request.Request{CMD: request.HEARTBEAT, OBJ: int64(10), URL: "127.0.0.1:5001"}
	response := client.Send(req)

	resutl := &request.HeartbeatResponse{}
	fmt.Println(string(response.Data.([](byte))))

	json.Unmarshal(response.Data.([]byte), resutl)
	fmt.Println(resutl)
}
