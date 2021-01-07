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

func NewRpcClient() *RpcClient {
	return &RpcClient{}
}

//func (r *RpcClient) Send(req *request.Request) *request.Response {
//	//"127.0.0.1:8096"
//	client, err := rpc.DialHTTP("tcp", req.URL)
//	if err != nil {
//		log.Fatal("dialing:", err)
//	}
//	var response request.Response
//	err = client.Call("RpcServer.HandleClientRequest", req, &response)
//	if err != nil {
//		fmt.Println("rpc call failed,because of: ", err)
//	}
//	//fmt.Println(response)
//
//	return &response
//
//}

func (r *RpcClient) Send(req *request.Request) *request.Response {
	//service := "127.0.0.1:1234"
	client, err := rpc.Dial("tcp", req.URL)
	if req.CMD == request.HEARTBEAT {
		fmt.Printf("heartbeat rpcclient send request to %v \n", req.URL)
	}
	defer client.Close()
	if err != nil {
		log.Fatal("dialing:", err)
	}
	response := &request.Response{}
	err = client.Call("RpcServer.HandleClientRequest", req, response)
	if err != nil {
		log.Fatal("HandleClientRequest error :", err)
	}
	//fmt.Println(response)
	return response
}
