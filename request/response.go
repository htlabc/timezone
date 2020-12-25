package request

const (
	OK     = "OK"
	FAILED = "FAILED"
)

type Response struct {
	Data interface{}
	Err  error
}
