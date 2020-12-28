package rpcserver

import (
	"fmt"
	req "htl.com/request"
	"htl.com/server"
	"net"
	"net/rpc"
	"os"
)

type RpcServer interface {
	Start()
	HandleClientRequest(request *req.Request, response *req.Response)
	Stop()
}

func NewRpcServer() *rpcserver {
	return &rpcserver{}
}

type Service interface {
	GetService() *server.Server
}

type rpcserver struct {
	Service
}

func (r *rpcserver) Start(address string, stopch chan struct{}) {
	func() {
		rpc.Register(r)
		tcpAddr, err := net.ResolveTCPAddr("tcp", address)
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

func (r *rpcserver) Stop() {

}

func (r *rpcserver) HandleClientRequest(request *req.Request, response *req.Response) {
	ser := r.Service.GetService()
	switch request.CMD {
	case req.V_ELECTION:
		response = ser.HandleElectionRequest(request)
	case req.G_TIMESTAMP:
		response = ser.HandleGetTimeZoneRequest(request)
	case req.A_PEER:
		response = ser.HandleAddPeerRequest(request)
	case req.R_PEER:
		response = ser.HandleRemovePeerRequest(request)
	case req.HEARTBEAT:
		response = ser.HandleHeartBeatRequest(request)
	}
}
