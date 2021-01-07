package rpcserver

import "testing"

func Test_rpcserver(t *testing.T) {
	rpcserver := &RpcServer{}
	stopch := make(chan struct{})
	Start(rpcserver, ":5001", stopch)
}
