package channelpool

import (
	"context"
	"errors"
	"sync"
	"time"
)

var callIdLock sync.Mutex
var callableIdMap map[int64]*Result

type Result struct {
	data interface{}
	err  error
}

type Callable struct {
	id     int64
	args   interface{}
	fun    func(interface{}) interface{}
	ctx    context.Context
	period time.Duration
	result *Result
}

func NewCallable(fun func(args interface{}) interface{}, args interface{}, ctx context.Context, period time.Duration) *Callable {
	return &Callable{fun: fun, ctx: ctx, period: period, args: args}
}

func getCallableId() int64 {
	callIdLock.Lock()
	defer callIdLock.Unlock()
	return time.Now().Unix()
}

type ChannelPool struct {
	CallableChan chan *Callable
	ExecuteChan  chan *Callable
	Size         int
	Cap          int
}

var Instance *ChannelPool

func GetInstace() *ChannelPool {
	if Instance != nil {
		return Instance
	} else {
		Instance = newChannelPool()
	}
	return Instance
}

func newChannelPool() *ChannelPool {
	pool := &ChannelPool{
		CallableChan: make(chan *Callable, 10),
		ExecuteChan:  make(chan *Callable, 10),
		Size:         10,
		Cap:          15,
	}
	go pool.producter()
	go pool.distribute()
	return pool
}

func (c *ChannelPool) producter() {
	for val := range c.CallableChan {
		if val.id == 0 {
			val.id = getCallableId()
		}
		c.ExecuteChan <- val
	}
}

func (c *ChannelPool) distribute() {
	for call := range c.ExecuteChan {
		go call.execute()
	}
}

//同步执行函数
func (c *ChannelPool) RunSync(calls []*Callable) []interface{} {
	var wg *sync.WaitGroup = new(sync.WaitGroup)
	results := make([]interface{}, 0)
	for _, call := range calls {
		wg.Add(1)
		func(c *Callable) {
			result := c.fun(c.args)
			if c.result == nil {
				c.result = &Result{}
			}
			c.result.data = result
			wg.Done()
		}(call)
	}
	wg.Wait()
	return results
}

//异步执行函数
func (c *ChannelPool) RunASync(calls []*Callable) {
	for _, call := range calls {
		c.CallableChan <- call
	}
}

func (c *Callable) execute() {
	ctx, _ := context.WithTimeout(c.ctx, c.period)
	var res interface{}
	func() {
		res = c.fun(c.args)
		switch res.(type) {
		case error:
			c.result.err = errors.New(res.(string))
		default:
			if res != nil {
				c.result.data = res
				c.result.err = nil
			}

		}
	}()
	<-ctx.Done()
	//超时管理
}

//获取协程池执行结果
func (pool *ChannelPool) Get(c *Callable) *Result {
	//<-c.ctx.Done()

	if c.result == nil {
		c.result = &Result{}
		return c.result
	}
	return c.result
}

func (r *Result) GetData() interface{} {
	if r != nil {
		return r.data
	}
	return nil
}

func (r *Result) GetError() error {
	return r.err
}
