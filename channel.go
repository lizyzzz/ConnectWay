package ConnectWay

import "net"

type MsgCallback func(msg *Message)
type ErrorCallback func(err error)

type Channel struct {
	msgCb        MsgCallback
	errCb        ErrorCallback
	closed       bool
	conn         net.Conn
	reqContainer *RequestContainer
	errorChannel chan error
	readChannel  chan *Message
	writeChannel chan *Message
	quitChannel  chan int32
}

func CreateChannel(Conn net.Conn, MsgCb MsgCallback, ErrCb ErrorCallback) *Channel {
	c := &Channel{
		msgCb:        MsgCb,
		errCb:        ErrCb,
		closed:       false,
		conn:         Conn,
		reqContainer: CreateRequestContainer(),
		errorChannel: make(chan error, 2),
		readChannel:  make(chan *Message, 128),
		writeChannel: make(chan *Message, 128),
		quitChannel:  make(chan int32),
	}
	return c
}

func (c *Channel) start() {
	go c.sendingLoop()
	go c.readingLoop()

outLoop:
	for {
		select {
		case msg := <-c.readChannel:
			if !c.reqContainer.FilterMessage(msg) {
				c.msgCb(msg)
			}
		case err := <-c.errorChannel:
			c.errCb(err)
			break outLoop
		}
	}
	c.Close()
}

func (c *Channel) sendingLoop() {
	for {
		select {
		case msg := <-c.writeChannel:
			// send header
			headerSend := 0
			for headerSend < 16 {
				n, err := c.conn.Write(msg.header[headerSend:])
				if err != nil {
					c.errorChannel <- err
					return
				}
				headerSend += n
			}

			// send payload
			byteLeft := msg.Length()
			byteTotal := byteLeft
			for byteLeft > 0 {
				n, err := c.conn.Write(msg.payload[byteTotal-byteLeft:])
				if err != nil {
					c.errorChannel <- err
					return
				}
				byteLeft -= uint32(n)
			}
		case <-c.quitChannel:
			// TODO: add log
			return
		}
	}
}

func (c *Channel) readingLoop() {
	for {
		// read header
		msg := &Message{}
		headerRead := 0
		for headerRead < 16 {
			n, err := c.conn.Read(msg.header[headerRead:])
			if err != nil {
				c.errorChannel <- err
				return
			}
			headerRead += n
		}

		// read payload
		msglen := msg.Length()
		msg.payload = make([]byte, msglen)
		var totalRead uint32 = 0
		for totalRead < msglen {
			n, err := c.conn.Read(msg.payload[totalRead:])
			if err != nil {
				c.errorChannel <- err
				return
			}
			totalRead += uint32(n)
		}

		c.readChannel <- msg
	}
}

func (c *Channel) sendMsg(msg *Message) {
	c.writeChannel <- msg
}

func (c *Channel) Close() {
	if !c.closed {
		c.closed = true
		c.conn.Close()
		c.quitChannel <- 1
	}
}
