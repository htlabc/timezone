package request

type HeartBeatRequest struct {
	Serverid string
	Term     int64
}
