package request

type HeartbeatResponse struct {
	Serverid string
	Term     int64
	Data     interface{}
}
