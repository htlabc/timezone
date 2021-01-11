package request

type HeartBeatRequest struct {
	Serverid  string
	Term      int64
	Timestamp int64
}
