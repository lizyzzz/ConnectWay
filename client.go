package ConnectWay

import (
	"net"
	"time"
)

type ConnectedCallback func(remote string)

type Client struct {
	connectedCb ConnectedCallback
	msgCb       MsgCallback
	errCb       ErrorCallback
	msgChannel  *Channel
	closed      bool
	serverAddr  string
	syncChan    chan int32
}

func CreateClient(ConnCb ConnectedCallback, MsgCb MsgCallback, ErrCb ErrorCallback) *Client {
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
			// TODO: add log
			return
		}
		c.msgChannel = CreateChannel(conn, c.msgCb, c.errCb)
		c.connectedCb(c.serverAddr)
		if !isAsync && c.closed {
			// avoid reconnect would send again
			c.syncChan <- 1
		}
		c.closed = false
		c.msgChannel.start()
		// reconnect
		time.Sleep(5 * time.Second)
		if c.closed {
			return
		}
	}
}

// TODO: sync and async (now is async)
func (c *Client) ConnectAsyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	// TODO: add log
	go c.start(true)
}

func (c *Client) ConnectSyncTo(address string) {
	if address != c.serverAddr {
		c.serverAddr = address
		c.Close()
	}
	// TODO: add log
	go c.start(false)
	<-c.syncChan // wait
}

func (c *Client) Close() {
	if !c.closed {
		c.closed = true
		c.msgChannel.Close()
	}
}
