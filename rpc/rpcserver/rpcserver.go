package rpcserver

import (
	"fmt"
	req "htl.com/request"
	"htl.com/server"
	"net"
	"net/rpc"
	"os"
)

type rpcServer interface {
	Start()
	HandleClientRequest(request *req.Request, response *req.Response)
	Stop()
}

func NewRpcServer() *RpcServer {
	return &RpcServer{}
}

//type Service interface {
//	GetService() *server.Server
//}

type RpcServer struct {
	//Service
}

//func Start(r *RpcServer, address string, stopch chan struct{}) {
//		rpc.Register(r)
//		//addr:=strings.Split(address,":")[1]
//		tcpAddr, err := net.ResolveTCPAddr("tcp", ":5001")
//		if err != nil {
//			fmt.Println("Fatal error:", err)
//			os.Exit(1)
//		}
//		listener, err := net.ListenTCP("tcp", tcpAddr)
//		if err != nil {
//			fmt.Println("Fatal error:", err)
//			os.Exit(1)
//		}
//
//		for {
//			conn, err := listener.Accept()
//			if err != nil {
//				continue
//			}
//			rpc.ServeConn(conn)
//		}
//}

func Start(r *RpcServer, address string, stopch chan struct{}) {

	rpc.Register(r)

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":1234")
	if err != nil {
		fmt.Println("Fatal error:", err)
		os.Exit(1)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Fatal error:", err)
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		rpc.ServeConn(conn)
	}
}

func Stop(r *RpcServer) {

}

func (r *RpcServer) HandleClientRequest(request *req.Request, response *req.Response) error {
	ser := server.GETINSTANCE()
	switch request.CMD {
	case req.V_ELECTION:
		ser.HandleElectionRequest(request, response)
	case req.G_TIMESTAMP:
		ser.HandleGetTimeZoneRequest(request, response)
	case req.A_PEER:
		ser.HandleAddPeerRequest(request, response)
	case req.R_PEER:
		ser.HandleRemovePeerRequest(request, response)
	case req.HEARTBEAT:
		ser.HandleHeartBeatRequest(request, response)
	}
	return nil
}
