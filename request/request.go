package request

var (
	G_TIMESTAMP = "GET_TIMESTAMP"
	V_ELECTION  = "VOTE"
	A_PEER      = "ADD_PEER"
	R_PEER      = "REMOVE_PEER"
	HEARTBEAT   = "HEARTBEAT"
)

type Request struct {
	CMD string
	OBJ interface{}
	URL string
}
