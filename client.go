package ConnectWay

import (
	"net"
	"time"

	log "github.com/cihub/seelog"
)

func init() {
	testConfig := `
	<seelog type="sync">
		<outputs formatid="common">
			<console/>
			<file path="./test.log"/>
		</outputs>
		<formats>
			<format id="common" format="[%Date(2006-01-02/15:04:05.000):%LEVEL:%File(%Line)] %Msg%n"/>
		</formats>
	</seelog>`
	logger, _ := log.LoggerFromConfigAsBytes([]byte(testConfig))
	log.ReplaceLogger(logger)
}

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

func CreateClient(ConnCb ConnectedCallback, MsgCb ClientMsgCallback, ErrCb ClientErrCallback) *Client {
	return &Client{
		connectedCb: ConnCb,
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
			return
		}
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
		log.Warnf("Disconnect %s, reconnect in 5 seconds", c.serverAddr)
		time.Sleep(5 * time.Second)
		if c.closed {
			return
		}
	}
}

func (c *Client) ConnectAsyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	go c.start(true)
}

func (c *Client) ConnectSyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	go c.start(false)
	<-c.syncChan // wait
}

func (c *Client) Close() {
	if !c.closed {
		c.closed = true
		c.msgChannel.Close()
	}
}

func (c *Client) SendMsg(m *Message) {
	c.msgChannel.sendMsg(m)
}

func (c *Client) Send(msgType uint32, msgLength uint32, seq uint32, msg []byte) {
	m := CreateMessage(msgType, msgLength, seq, msg)
	c.SendMsg(m)
}

func (c *Client) Request(ReqType uint32, msgLength uint32, seq uint32, msg []byte, RespCb RespCallback) {
	req := CreateRequest(ReqType, RespCb)
	c.msgChannel.reqContainer.AddRequest(req)
	m := CreateMessage(ReqType, msgLength, seq, msg)
	c.SendMsg(m)
}

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
