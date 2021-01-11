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
	"math/rand"
	"os"
	"strconv"
	"strings"
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
	HandleGetTimeZoneRequest(request *request.Request, response *request.Response)
	HandleAddPeerRequest(request *request.Request, response *request.Response)
	HandleRemovePeerRequest(request *request.Request, response *request.Response)
	HandleElectionRequest(request *request.Request, response *request.Response)
	Redirect(request *request.Request, response *request.Response)
	HandleHeartBeatRequest(req *request.Request, response *request.Response)
}

//server参数
type Option struct {
}

type Server struct {
	peerset              PeerSet
	votefor              string
	rpcclient            *rpcclient.RpcClient
	leaderid             string
	status               string
	serverid             *string
	term                 int64
	preelectiontimestamp int64
	electionTime         int64
	electionperiod       int
	heartbeatperiod      int64
	preHeartbeatTime     int64
	detaltime            int64
	retryPeriod          int
}

func NewServer() *Server {
	return &Server{rpcclient: &rpcclient.RpcClient{}, heartbeatperiod: 5, electionperiod: 3, status: FLLOWER, term: 1}
}

func GETINSTANCE() *Server {
	if INSTANCE != nil {
		return INSTANCE
	} else {
		return NewServer()
	}
}

func (s *Server) Start(stopch chan struct{}) error {
	serverid := consensus.Hash([]byte(s.GetSelfPeer().address))
	s.serverid = &serverid
	fmt.Printf("server %v begin to start \n", s.GetSelfPeer().address)

	termstr, err := s.Read(consensus.DataPath + *s.serverid)
	if err != nil {
		return err
	}
	term := strings.Split(string(termstr), "-")[1]
	d, _ := strconv.Atoi(term)
	num := (*int64)(unsafe.Pointer(&d))
	s.term = *num
	go s.ElectionScheduler(time.NewTicker(3 * time.Second))
	go s.heartbeat(time.NewTicker(500 * time.Millisecond))
	<-stopch
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
	for f := s.peerset.list.Front(); f != nil; f = f.Next() {
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

func (s *Server) ElectionScheduler(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			func() {
				if s.status == LEADER {
					return
				}
				currenttime := consensus.MakeTimestamp(time.Now())
				s.electionTime = s.electionTime + int64(rand.Intn(50))
				//如果投票间隔的时间比当前时间减去上一次投票时间还要大则返回不产生投票
				if currenttime-s.preelectiontimestamp < s.electionTime {
					return
				}
				////超时需要选举投票
				//if s.preleaderheartbeat.Add(time.Duration(s.heartbeatperiod)*time.Second).Before(currenttime) || s.votefor == "" {
				//	fmt.Printf("watch leader %v could be over will election \n", s.leaderid)
				//} else {
				//	return
				//}
				s.preelectiontimestamp = currenttime + int64(rand.Intn(200)) + 150
				s.status = CANDIDATE
				s.votefor = *s.serverid
				s.term += 1
				var tickets int32 = 0
				callables := make([]*channelpool.Callable, 0)
				for _, f := range s.GetWithoutSelfPeer() {
					callables = append(callables, channelpool.NewCallable(func(args interface{}) interface{} {
						p := args.(Peer)
						fmt.Println("peer address: ", p.address)
						votereq := &request.VoteRequest{*s.serverid, s.term}
						jsdat, _ := json.Marshal(votereq)
						req := &request.Request{CMD: request.V_ELECTION, URL: p.address, OBJ: jsdat}
						result := s.rpcclient.Send(req)
						return result
					}, f, context.Background(), 5*time.Second))
				}
				channelpool.Instance.RunSync(callables)
				for _, c := range callables {
					jsdata := channelpool.GetInstace().Get(c).GetData().(*request.Response).Data
					response := &request.VoteResponse{}
					json.Unmarshal(jsdata.([]byte), response)
					if response.Term > s.term {
						s.term = response.Term
						//把term写入文件保存防止异常宕机之后信息丢失
						//s.Write([]byte(*s.serverid + "-" + strconv.FormatInt(s.term, 10)))
						//s.leaderid = response.Serverid
						return
					}
					if response.Data.(string) == request.OK {
						atomic.AddInt32(&tickets, 1)
						//tickets++
					}
				}

				//如果接收到来自新的领导人的心跳RPC，导致转变成跟随者直接退出流程
				if s.status == FLLOWER {
					return
				}
				if tickets >= int32(len(s.GetWithoutSelfPeer()))/2 {
					fmt.Printf("become to leader %v \n", *s.serverid)
					s.status = LEADER
					s.leaderid = *s.serverid
				} else {
					s.votefor = ""
				}

			}()
		}
	}

}

//t := time.NewTicker(time.Duration(s.heartbeatperiod) * time.Second)
func (s *Server) heartbeat(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			func() {
				if s.status != LEADER {
					return
				}
				s.Write([]byte(*s.serverid + "-" + strconv.FormatInt(s.term, 10)))
				callables := make([]*channelpool.Callable, 0)
				fmt.Printf("heartbeat server address: %v server status %v leaderid %v term is %v  votedfor %v \n", s.GetSelfPeer().address, s.status, s.leaderid, s.term, s.votefor)
				currenttime := consensus.MakeTimestamp(time.Now())
				if currenttime-s.preHeartbeatTime <= s.heartbeatperiod {
					return
				}
				s.preHeartbeatTime = consensus.MakeTimestamp(time.Now())
				for _, f := range s.GetWithoutSelfPeer() {
					callables = append(callables, channelpool.NewCallable(
						func(args interface{}) interface{} {
							switch args.(type) {
							case Peer:
							default:
								return request.Response{}
							}
							hbreq := &request.HeartBeatRequest{*s.serverid, s.term, consensus.MakeTimestamp(time.Now())}
							jsdata, _ := json.Marshal(hbreq)
							req := &request.Request{CMD: request.HEARTBEAT, OBJ: jsdata, URL: args.(Peer).address}
							response := s.rpcclient.Send(req)
							hbresp := &request.HeartbeatResponse{}
							json.Unmarshal(response.Data.([]byte), hbresp)
							if hbresp.Term > s.term {
								s.term = hbresp.Term
								s.status = FLLOWER
							}

							return nil
						}, f, context.Background(), 5*time.Second))

				}
				channelpool.GetInstace().RunASync(callables)

			}()
		}
	}

}

func (s *Server) HandleHeartBeatRequest(req *request.Request, response *request.Response) {
	hbreq := &request.HeartBeatRequest{}
	json.Unmarshal(req.OBJ.([]byte), hbreq)
	if s.detaltime == 0 {
		s.detaltime = consensus.MakeTimestamp(time.Now()) - hbreq.Timestamp
	}
	fmt.Printf("server: %v status: %v  term: %v recive serverid  %v heartbeat term is %v \n", *s.serverid, s.status, s.term, hbreq.Serverid, hbreq.Term)
	//如果leader发过来的term任期比自己大则接受它的心跳否则就返回
	if s.term <= hbreq.Term {
		s.leaderid = hbreq.Serverid
		s.status = FLLOWER
		s.term = hbreq.Term
		s.Write([]byte(*s.serverid + "-" + strconv.FormatInt(s.term, 10)))
	} else {
		jsdata, _ := json.Marshal(request.HeartbeatResponse{*s.serverid, s.term, request.FAILED})
		response.Data = jsdata
		return
	}
	s.preHeartbeatTime = consensus.MakeTimestamp(time.Now())
	s.preelectiontimestamp = consensus.MakeTimestamp(time.Now())
	jsdata, _ := json.Marshal(request.HeartbeatResponse{*s.serverid, s.term, request.OK})
	response.Data = jsdata
}

func (s *Server) HandleElectionRequest(req *request.Request, response *request.Response) {
	var votelock sync.Mutex
	votelock.Lock()
	defer votelock.Unlock()
	vq := req.OBJ
	voterequest := &request.VoteRequest{}
	json.Unmarshal(vq.([]byte), voterequest)
	vreq := &request.VoteResponse{*s.serverid, s.term, request.OK}
	//如果对方任期没有自己的新直接返回错误
	if s.term > voterequest.Term {
		vreq.Data = request.FAILED
		data, _ := json.Marshal(vreq)
		response.Data = data
		response.Err = nil
		return
	}

	//当第一次投票的时候votefor是为空，当节点已经投票后下次还是会投给上一次的节点
	if s.votefor == "" || s.votefor == voterequest.Serverid {
		//fmt.Printf("begin to HandleElectionRequest recive serverid %v, term is %v \n", voterequest.Serverid, voterequest.Term)
		s.votefor = voterequest.Serverid
		s.leaderid = voterequest.Serverid
		s.term = voterequest.Term
		s.status = FLLOWER
		data, _ := json.Marshal(vreq)
		response.Data = data
		response.Err = nil
		return
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
	if s.leaderid != *s.serverid {
		s.Redirect(req, response)
		return
	}
	var lock sync.Mutex
	lock.Lock()
	defer lock.Unlock()
	response.Data = time.Now().Nanosecond()
	response.Err = nil
}

func (s *Server) GetTimeZone() *time.Time {
	fmt.Println("begin to GetTimeZone")
	if s == nil {
		return nil
	}
	req := &request.Request{request.G_TIMESTAMP, nil, ""}
	var response *request.Response
	if *s.serverid != s.leaderid {
		s.Redirect(req, response)
	}
	if response == nil {
		return nil
	}
	time := response.Data.(time.Time)
	return &time
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
	if s.peerset.leader == nil {
		return
	}
	request.URL = s.peerset.leader.address
	response = s.rpcclient.Send(request)
}

func (s *Server) Write(data []byte) {
	fmt.Println("begin to write head log \n", consensus.DataPath+*s.serverid)
	fi, err := os.Open(consensus.DataPath + *s.serverid)
	defer fi.Close()
	if err != nil && os.IsNotExist(err) {
		return
	}
	file, err := os.Create(consensus.DataPath + *s.serverid)
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

func (s *Server) RequestAddPeer(p *Peer) bool {
	var callables []*channelpool.Callable = make([]*channelpool.Callable, 0)
	retrynum := 0
ADDPEER_HERE:
	if retrynum == s.retryPeriod {
		return false
	}
	for _, peer := range s.GetWithoutSelfPeer() {
		callables = append(callables, channelpool.NewCallable(func(args interface{}) interface{} {
			req := request.Request{CMD: request.A_PEER, OBJ: p, URL: peer.address}
			result := s.rpcclient.Send(&req)
			return result
		}, nil, context.Background(), 5*time.Second))
	}

	channelpool.GetInstace().RunSync(callables)

	success := 0
	for _, cal := range callables {
		result := channelpool.GetInstace().Get(cal)
		if result.GetData().(string) == request.OK {
			success += 1
		}
	}
	if success == len(s.GetWithoutSelfPeer()) {
		return true
	} else {
		retrynum++
		goto ADDPEER_HERE
	}

	return false
}
