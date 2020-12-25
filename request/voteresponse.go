package request

type VoteResponse struct {
	Serverid string
	Term     int64
	Data     interface{}
}
