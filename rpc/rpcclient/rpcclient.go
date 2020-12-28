package rpcclient

import (
	"fmt"
	"htl.com/request"
	"log"
	"net/rpc"
)

type rpcclient interface {
	Send(req *request.Request) *request.Response
}

type RpcClient struct {
}

func (r *RpcClient) Send(req *request.Request) *request.Response {
	//"127.0.0.1:8096"
	conn, err := rpc.DialHTTP("tcp", req.URL)
	if err != nil {
		log.Fatalln()
	}
	var response request.Response
	err = conn.Call("RpcServer.HandleClientRequest", req, &response)
	if err != nil {
		fmt.Println("rpc call failed,because of: ", err)
	}
	return &response

}
