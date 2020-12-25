package rpcserver

import (
	"fmt"
	req "htl.com/request"
	"htl.com/server"
	"net"
	"net/rpc"
	"os"
)

type rpcserver interface {
	Start()
	HandleClientRequest(request *req.Request, response *req.Response)
	Stop()
}

type RpcServer struct {
}

func (r *RpcServer) Start() {
	go func() {
		rpc.Register(r)
		tcpAddr, err := net.ResolveTCPAddr("tcp", ":1234")
		if err != nil {
			fmt.Println("Fatal error:", err)
			os.Exit(1)
		}
		listener, err := net.ListenTCP("tcp", tcpAddr)
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			rpc.ServeConn(conn)
		}
	}()
}

func (r *RpcServer) Stop() {

}

func (r *RpcServer) HandleClientRequest(request *req.Request, response *req.Response) {
	server := server.GETINSTANCE()
	switch request.CMD {
	case req.V_ELECTION:
		response = server.HandleElectionRequest(request)
	case req.G_TIMESTAMP:
		response = server.HandleGetTimeZoneRequest(request)
	case req.A_PEER:
		response = server.HandleAddPeerRequest(request)
	case req.R_PEER:
		response = server.HandleRemovePeerRequest(request)
	case req.HEARTBEAT:
		response = server.HandleHeartBeatRequest(request)
	}
}
