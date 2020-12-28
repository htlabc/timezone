package server

import (
	"container/list"
	"context"
	"fmt"
	"htl.com/channelpool"
	"htl.com/request"
	"htl.com/rpc/rpcclient"
	"htl.com/util"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var INSTANCE *Server

const (
	FLLOWER   = "FLLOWER"
	LEADER    = "LEADER"
	CANDIDATE = "CANDIDATE"
)

type server interface {
	Start()
	Stop()
}

type Service interface {
	HandleGetTimeZoneRequest(request *request.Request) *request.Response
	HandleAddPeerRequest(request *request.Request) *request.Response
	HandleRemovePeerRequest(request *request.Request) *request.Response
	HandleElectionRequest(request *request.Request) *request.Response
	Redirect(request *request.Request) *request.Response
	HandleHeartBeatRequest(req *request.Request) *request.Response
}

//server参数
type Option struct {
}

type Server struct {
	peerset         PeerSet
	votefor         string
	rpcclient       *rpcclient.RpcClient
	leaderid        string
	status          string
	serverid        string
	term            int64
	pretimestamp    time.Time
	timeoutperiod   int
	heartbeatperiod int
}

func NewServer() *Server {
	return &Server{rpcclient: &rpcclient.RpcClient{}}
}

func GETINSTANCE() *Server {
	if INSTANCE != nil {
		return INSTANCE
	} else {
		return NewServer()
	}
}

func (s *Server) Start(stopch chan struct{}) error {
	serverid := consensus.Hash(s.GetSelfPeer().address)
	s.serverid = string(serverid)
	term, err := s.Read(consensus.DataPath + s.serverid)
	if err != nil {
		return err
	}
	d, _ := strconv.Atoi(string(term))
	num := (*int64)(unsafe.Pointer(&d))
	s.term = *num
	go s.electionScheduler(time.NewTicker(5 * time.Second))
	go s.heartbeat(time.NewTicker(5 * time.Second))
	return nil
}

func (s *Server) GetService() *Server {
	return s
}

func (s *Server) GetSelfPeer() *Peer {
	return s.peerset.self
}

func (s *Server) GetWithoutSelfPeer() []Peer {
	var peers = make([]Peer, 0)
	for f := s.peerset.list.Front(); f != nil; f.Next() {
		if f.Value.(Peer).address == s.peerset.self.address {
			continue
		}
		peers = append(peers, f.Value.(Peer))
	}

	return peers
}

func (s *Server) SetPeerSet(peers []Peer, self *Peer) {
	s.peerset.list = list.New()
	for peer := range peers {
		s.peerset.list.PushBack(peer)
	}
	s.peerset.self = self
}

func (s *Server) electionScheduler(ticker *time.Ticker) {
	select {
	case <-ticker.C:
		go func() {
			currenttime := time.Now()
			//超时需要选举投票
			if s.pretimestamp.Add(time.Duration(s.timeoutperiod) * time.Second).Before(currenttime) {
				s.status = CANDIDATE
				s.term += 1
				var tickets int32 = 0
				callables := make([]*channelpool.Callable, 0)
				for f := s.peerset.list.Front(); f != nil; f = f.Next() {
					callables = append(callables, channelpool.NewCallable(
						func(args interface{}) interface{} {
							p := args.(Peer)
							votereq := &request.VoteRequest{s.serverid, s.term}
							req := &request.Request{CMD: request.V_ELECTION, URL: p.address, OBJ: votereq}
							return s.rpcclient.Send(req)
						}, f.Value.(Peer), context.Background(), 5*time.Second,
					))
				}
				channelpool.Instance.RunSync(callables)
				for _, c := range callables {
					response := channelpool.GetInstace().Get(c).GetData().(request.VoteResponse)
					if response.Term > s.term {
						s.term = response.Term
						//把term写入文件保存防止异常宕机之后信息丢失
						s.Write([]byte(s.serverid + "-" + strconv.FormatInt(s.term, 10)))
						s.status = FLLOWER
						s.leaderid = response.Serverid
						return
					}
					if response.Data.(string) == request.OK {
						atomic.AddInt32(&tickets, 1)
					}
				}

				if int32(len(s.GetWithoutSelfPeer())) > tickets/2 {
					s.status = LEADER
					s.leaderid = s.serverid
				}

			}

		}()
	}

}

//t := time.NewTicker(time.Duration(s.heartbeatperiod) * time.Second)
func (s *Server) heartbeat(ticker *time.Ticker) {
	select {
	case <-ticker.C:
		go func() {
			callables := make([]*channelpool.Callable, 0)
			for f := s.peerset.list.Front(); f != nil; f = f.Next() {
				callables = append(callables, channelpool.NewCallable(
					func(args interface{}) interface{} {
						req := &request.Request{CMD: request.HEARTBEAT, OBJ: s.term, URL: args.(Peer).address}
						response := s.rpcclient.Send(req)
						if response.Data.(request.HeartbeatResponse).Serverid == s.leaderid {
							s.pretimestamp = response.Data.(request.HeartbeatResponse).Data.(time.Time)
						}
						return response
					}, f, context.Background(), 5*time.Second))

			}
			channelpool.GetInstace().RunASync(callables)
		}()
	}

}

func (s *Server) HandleHeartBeatRequest(req *request.Request) *request.Response {
	if s.term > req.OBJ.(int64) {
		s.leaderid = s.leaderid
		s.status = FLLOWER
		s.term = req.OBJ.(int64)
		s.Write([]byte(s.serverid + "-" + strconv.FormatInt(s.term, 10)))
	}
	response := &request.Response{}
	response.Data = &request.HeartbeatResponse{s.serverid, s.term, time.Now()}
	return response
}

func (s *Server) HandleElectionRequest(req *request.Request) *request.Response {
	var votelock sync.Mutex
	votelock.Lock()
	defer votelock.Unlock()
	voterequest := req.OBJ.(request.VoteRequest)
	vreq := &request.VoteResponse{s.serverid, s.term, request.OK}
	if s.votefor == "" || s.votefor == voterequest.Serverid {
		peer := s.peerset.leader
		s.votefor = voterequest.Serverid
		s.leaderid = voterequest.Serverid
		s.term = voterequest.Term
		s.status = FLLOWER
		if peer != nil {
			return &request.Response{vreq, nil}
		}
	} else {
		vreq.Data = request.FAILED
		return &request.Response{vreq, nil}
	}
	return nil
}

//处理timezone请求
func (s *Server) HandleGetTimeZoneRequest(req *request.Request) *request.Response {
	//如果不是leader 则转发给leader 执行请求
	if s.leaderid != s.serverid {
		return s.Redirect(req)
	}
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	response := &request.Response{time.Now().Nanosecond() + rand.Intn(10000), nil}
	return response
}

func (s *Server) GetTimeZone() time.Time {
	req := &request.Request{request.G_TIMESTAMP, nil, ""}
	var response *request.Response
	if s.serverid != s.leaderid {
		response = s.Redirect(req)
	}
	return response.Data.(time.Time)
}

func (s *Server) HandleAddPeerRequest(req *request.Request) *request.Response {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	s.peerset.list.PushBack(req.OBJ.(Peer))
	return &request.Response{request.OK, nil}
}

func (s *Server) HandleRemovePeerRequest(req *request.Request) *request.Response {
	for f := s.peerset.list.Front(); f != nil; f = f.Next() {
		if f.Value.(Peer).address == req.OBJ.(Peer).address {
			s.peerset.list.Remove(f)
			return &request.Response{request.OK, nil}
		}
	}
	return &request.Response{request.FAILED, nil}
}

func (s *Server) Redirect(request *request.Request) *request.Response {
	request.URL = s.peerset.leader.address
	res := s.rpcclient.Send(request)
	return res
}

func (s *Server) Write(data []byte) {
	file, err := os.Create(consensus.DataPath + s.serverid)
	defer file.Close()
	if err != nil {
		fmt.Printf("os.create file err %v", err)
		return
	}
	file.Write(data)
}

func (s *Server) Read(filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil
	}
	return data, nil
}
