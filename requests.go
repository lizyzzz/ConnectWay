package ConnectWay

import (
	"fmt"
	"sync"
	"time"
)

type RespCallback func(msg *Message, err error)

type Request struct {
	seq      uint32
	respType uint32
	respCb   RespCallback
}

type RequestContainer struct {
	lock     sync.Mutex
	requests map[uint32]*Request
}

var seqCount uint32 = 0

func generateSeq() uint32 {
	seqCount++
	return seqCount
}

func CreateRequest(ReqType uint32, RespCb RespCallback) *Request {
	result := &Request{
		seq:      generateSeq(),
		respType: ReqType + 1, // respType == ReqType + 1
		respCb:   RespCb,
	}
	return result
}

func CreateRequestContainer() *RequestContainer {
	return &RequestContainer{
		requests: make(map[uint32]*Request, 32),
	}
}

func (rc *RequestContainer) AddRequest(req *Request) {
	rc.lock.Lock()
	rc.requests[req.respType] = req
	rc.lock.Unlock()
	// timeout
	// TODO: add time wheel to reduce gorountine
	go rc.timeoutCallback(req.respType)
}

func (rc *RequestContainer) FilterMessage(msg *Message) bool {
	msgType := msg.Type()
	req, ok := rc.requests[msgType]
	if ok {
		rc.lock.Lock()
		delete(rc.requests, msgType)
		rc.lock.Unlock()
		req.respCb(msg, nil)
		return true
	}
	return false
}

func (rc *RequestContainer) timeoutCallback(respType uint32) {
	// 10s timeout
	<-time.After(10 * time.Second)
	req, ok := rc.requests[respType]
	if ok {
		rc.lock.Lock()
		delete(rc.requests, respType)
		rc.lock.Unlock()
		// TODO: add log
		err := fmt.Errorf("request: %d timeout", respType)
		req.respCb(nil, err)
	}
}
