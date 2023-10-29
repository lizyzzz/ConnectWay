package ConnectWay

import (
	"net"
	"time"

	log "github.com/cihub/seelog"
)

type ConnectedCallback func(c *Client)
type ClientMsgCallback func(m *Message, c *Client)
type ClientErrCallback func(err error, c *Client)

type Client struct {
	connectedCb ConnectedCallback
	msgCb       ClientMsgCallback
	errCb       ClientErrCallback
	msgChannel  *Channel
	closed      bool
	serverAddr  string
	syncChan    chan int32
}

// Create a client with callback function
func CreateClient(ConnectCb ConnectedCallback, MsgCb ClientMsgCallback, ErrCb ClientErrCallback) *Client {
	return &Client{
		connectedCb: ConnectCb,
		msgCb:       MsgCb,
		errCb:       ErrCb,
		msgChannel:  nil,
		closed:      true,
		serverAddr:  "",
		syncChan:    make(chan int32),
	}
}

func (c *Client) start(isAsync bool) {
	for {
		conn, err := net.Dial("tcp", c.serverAddr)
		if err != nil {
			log.Errorf("connect %s failed, err: %s", c.serverAddr, err.Error())
			log.Warnf("reconnect %s in 5 seconds", c.serverAddr)
		} else {
			c.msgChannel = CreateChannel(conn, c.OnChannelMsg, c.OnChannelError)
			c.connectedCb(c)
			if !isAsync && c.closed {
				// avoid reconnect would send again
				c.syncChan <- 1
			}
			c.closed = false
			log.Infof("connect %s successed", c.serverAddr)
			c.msgChannel.start()
			// reconnect
			if c.closed {
				return
			}
			log.Warnf("Disconnect %s, reconnect in 5 seconds", c.serverAddr)
		}
		time.Sleep(5 * time.Second)
	}
}

// connet to address async
// need to make sure that is connected
func (c *Client) ConnectAsyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	go c.start(true)
}

// connet to address sync
func (c *Client) ConnectSyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	go c.start(false)
	<-c.syncChan // wait
}

// close connect
func (c *Client) Close() {
	if !c.closed {
		c.closed = true
		c.msgChannel.Close()
	}
}

// send message to server,
// need to package message
func (c *Client) SendMsg(m *Message) {
	c.msgChannel.sendMsg(m)
}

// send message to server with type,
// don't need to package message
func (c *Client) Send(msgType uint32, msgLength uint32, seq uint32, msg []byte) {
	m := CreateMessage(msgType, msgLength, seq, msg)
	c.SendMsg(m)
}

// make a request to server,
// need to make sure wait to reply
func (c *Client) Request(ReqType uint32, msgLength uint32, seq uint32, msg []byte, RespCb RespCallback) {
	req := CreateRequest(ReqType, RespCb)
	c.msgChannel.reqContainer.AddRequest(req)
	m := CreateMessage(ReqType, msgLength, seq, msg)
	c.SendMsg(m)
}

// reply a request
func (c *Client) Reply(ReplyType uint32, msgLength uint32, seq uint32, msg []byte) {
	m := CreateMessage(ReplyType+1, msgLength, seq, msg)
	c.SendMsg(m)
}

func (c *Client) OnChannelMsg(msg *Message) {
	c.msgCb(msg, c)
}

func (c *Client) OnChannelError(err error) {
	log.Errorf("on channel error: %v", err)
	c.errCb(err, c)
}

// get local address
func (c *Client) LocalAddr() net.Addr {
	return c.msgChannel.conn.LocalAddr()
}

// get remote address
func (c *Client) RemoteAddr() net.Addr {
	return c.msgChannel.conn.RemoteAddr()
}
