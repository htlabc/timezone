package server

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"htl.com/channelpool"
	"htl.com/request"
	"htl.com/rpc/rpcclient"
	"htl.com/util"
	"io/ioutil"
	"os"
	"reflect"
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
	for _, peer := range peers {
		s.peerset.list.PushFront(peer)
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
						switch args.(type) {
						case *list.Element:
							fmt.Println("args value: ", reflect.TypeOf(args))
						default:
							return request.Response{}
						}
						req := &request.Request{CMD: request.HEARTBEAT, OBJ: s.term, URL: args.(*list.Element).Value.(Peer).address}
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

func (s *Server) HandleHeartBeatRequest(req *request.Request, response *request.Response) {
	if s.term > req.OBJ.(int64) {
		s.leaderid = s.leaderid
		s.status = FLLOWER
		s.term = req.OBJ.(int64)
		s.Write([]byte(s.serverid + "-" + strconv.FormatInt(s.term, 10)))
	}
	jsdata, _ := json.Marshal(request.HeartbeatResponse{s.serverid, s.term, time.Now()})
	response.Data = jsdata
}

func (s *Server) HandleElectionRequest(req *request.Request, response *request.Response) {
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
			data, _ := json.Marshal(vreq)
			response.Data = data
			response.Err = nil
			return
		}
	} else {
		vreq.Data = request.FAILED
		data, _ := json.Marshal(vreq)
		response.Data = data
		response.Err = nil
		return
	}

}

//处理timezone请求
func (s *Server) HandleGetTimeZoneRequest(req *request.Request, response *request.Response) {
	//如果不是leader 则转发给leader 执行请求
	if s.leaderid != s.serverid {
		s.Redirect(req, response)
		return
	}
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	response.Data = time.Now().Nanosecond()
	response.Err = nil
}

func (s *Server) GetTimeZone() time.Time {
	req := &request.Request{request.G_TIMESTAMP, nil, ""}
	var response *request.Response
	if s.serverid != s.leaderid {
		s.Redirect(req, response)
	}
	return response.Data.(time.Time)
}

func (s *Server) HandleAddPeerRequest(req *request.Request, response *request.Response) {
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	s.peerset.list.PushBack(req.OBJ.(Peer))
	response.Data = request.OK
	response.Err = nil
}

func (s *Server) HandleRemovePeerRequest(req *request.Request, response *request.Response) {
	for f := s.peerset.list.Front(); f != nil; f = f.Next() {
		if f.Value.(Peer).address == req.OBJ.(Peer).address {
			s.peerset.list.Remove(f)
			response.Data = request.OK
			response.Err = nil
			return
		}
	}

	response.Data = request.FAILED
	response.Err = nil
}

func (s *Server) Redirect(request *request.Request, response *request.Response) {
	request.URL = s.peerset.leader.address
	response = s.rpcclient.Send(request)
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
