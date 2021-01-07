package rpcserver

import (
	"fmt"
	req "htl.com/request"
	"htl.com/server"
	"net"
	"net/rpc"
	"os"
	"strings"
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

func init() {
	ServerQueue = make(map[string]*server.Server, 0)
}

var ServerQueue map[string]*server.Server

type RpcServer struct {
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
	addr := strings.Split(address, ":")[1]
	fmt.Println("regiest rpc server port: ", addr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+addr)
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
	var ser *server.Server = ServerQueue[request.URL]
	fmt.Printf("server address %v recive request type %v \n", ser.GetSelfPeer().GetAddress(), request.CMD)
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
