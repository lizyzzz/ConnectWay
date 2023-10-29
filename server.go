package ConnectWay

import (
	"net"

	log "github.com/cihub/seelog"
)

type NewClientCallback func(stub *ClientStub)
type StubMsgCallback func(m *Message, stub *ClientStub)
type StubErrCallback func(err error, stub *ClientStub)

type ClientStub struct {
	msgCb      StubMsgCallback
	errCb      StubErrCallback
	msgChannel *Channel
	closed     bool
}

type Server struct {
	clientCb    NewClientCallback
	msgCb       StubMsgCallback
	errCb       StubErrCallback
	address     string
	listener    net.Listener
	clientStubs map[*ClientStub]bool
	stubChan    chan *ClientStub
	rmChan      chan *ClientStub
}

// Create a clientStub with net.Conn and callback function
func CreateClientStub(conn net.Conn, MsgCb StubMsgCallback, ErrCb StubErrCallback) *ClientStub {
	result := &ClientStub{
		msgCb:  MsgCb,
		errCb:  ErrCb,
		closed: false,
	}
	result.msgChannel = CreateChannel(conn, result.OnChannelMsg, result.OnChannelError)
	return result
}

// Create a server with address and callback function
func CreateServer(addr string, newClientCb NewClientCallback, MsgCb StubMsgCallback, ErrCb StubErrCallback) *Server {
	return &Server{
		clientCb:    newClientCb,
		msgCb:       MsgCb,
		errCb:       ErrCb,
		address:     addr,
		listener:    nil,
		clientStubs: make(map[*ClientStub]bool, 32),
		stubChan:    make(chan *ClientStub, 32),
		rmChan:      make(chan *ClientStub, 32),
	}
}

func (s *Server) Start() {
	var lisErr error
	s.listener, lisErr = net.Listen("tcp", s.address)
	if lisErr != nil {
		log.Errorf("Listen failed in %s", s.address)
		return
	}
	go s.handleClientStub()
	log.Infof("Server start, listen in: %s", s.address)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Errorf("Accept error: %v", err)
			continue
		}
		stub := CreateClientStub(conn, s.msgCb, s.errCb)
		s.stubChan <- stub
	}
}

func (s *Server) handleClientStub() {
	for {
		select {
		case stub := <-s.stubChan:
			s.clientStubs[stub] = true
			go s.handleStubConn(stub)
			s.clientCb(stub)
		case stub := <-s.rmChan:
			delete(s.clientStubs, stub)
		}
	}
}

func (s *Server) handleStubConn(stub *ClientStub) {
	stub.msgChannel.start()
	stub.Close()
	s.rmChan <- stub
}

func (stub *ClientStub) OnChannelMsg(msg *Message) {
	stub.msgCb(msg, stub)
}

func (stub *ClientStub) OnChannelError(err error) {
	log.Errorf("on channel error: %v", err)
	stub.errCb(err, stub)
}

// send message to client,
// need to package message
func (stub *ClientStub) SendMsg(msg *Message) {
	stub.msgChannel.sendMsg(msg)
}

// send message to server with type,
// don't need to package message
func (stub *ClientStub) Send(msgType uint32, msgLength uint32, seq uint32, msg []byte) {
	m := CreateMessage(msgType, msgLength, seq, msg)
	stub.SendMsg(m)
}

// make a request to client,
// need to make sure wait to reply
func (stub *ClientStub) Request(ReqType uint32, msgLength uint32, seq uint32, msg []byte, RespCb RespCallback) {
	req := CreateRequest(ReqType, RespCb)
	stub.msgChannel.reqContainer.AddRequest(req)
	m := CreateMessage(ReqType, msgLength, seq, msg)
	stub.SendMsg(m)
}

// reply a request
func (stub *ClientStub) Reply(ReplyType uint32, msgLength uint32, seq uint32, msg []byte) {
	m := CreateMessage(ReplyType+1, msgLength, seq, msg)
	stub.SendMsg(m)
}

// close connect
func (stub *ClientStub) Close() {
	if !stub.closed {
		stub.closed = true
		stub.msgChannel.Close()
	}
}

// get local address
func (stub *ClientStub) LocalAddr() net.Addr {
	return stub.msgChannel.conn.LocalAddr()
}

// get remote address
func (stub *ClientStub) RemoteAddr() net.Addr {
	return stub.msgChannel.conn.RemoteAddr()
}
