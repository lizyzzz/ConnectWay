package main

import (
	cy "ConnectWay"
	"fmt"
	"time"

	log "github.com/cihub/seelog"
)

func InitSeeLog() {
	testConfig := `
	<seelog type="sync">
		<outputs formatid="common">
			<console/>
			<file path="./client.log"/>
		</outputs>
		<formats>
			<format id="common" format="[%Date(2006-01-02/15:04:05.000):%LEVEL:%File(%Line)] %Msg%n"/>
		</formats>
	</seelog>`
	logger, _ := log.LoggerFromConfigAsBytes([]byte(testConfig))
	log.ReplaceLogger(logger)
}

const (
	MsgType    uint32 = 0x55
	ReqMsgType uint32 = 0x122
)

type MyClient struct {
	cli *cy.Client
}

func (c *MyClient) OnConnected(cli *cy.Client) {
	log.Infof("connected success, local: %s, remote: %s", cli.LocalAddr().String(), cli.RemoteAddr().String())
}

func (c *MyClient) OnMsg(msg *cy.Message, cli *cy.Client) {
	log.Infof("recv message: type: %d, payload: %s", msg.Type(), string(msg.Payload()))
}

func (c *MyClient) OnErr(err error, cli *cy.Client) {
	log.Infof("err: %s", err.Error())
}

func main() {
	InitSeeLog()

	c := &MyClient{}
	c.cli = cy.CreateClient(c.OnConnected, c.OnMsg, c.OnErr)
	c.cli.ConnectSyncTo("localhost:50051")

	for i := 0; i < 60; i++ {
		str := fmt.Sprintf("hello num: %d", i)
		data := []byte(str)
		c.cli.Send(MsgType, uint32(len(data)), uint32(i), data)
		time.Sleep(1 * time.Second)
	}

	for j := 0; j < 5; j++ {
		str := fmt.Sprintf("Request num: %d", j)
		data := []byte(str)
		waitchan := make(chan int32)
		c.cli.Request(ReqMsgType, uint32(len(data)), uint32(j), data,
			func(msg *cy.Message, err error) {
				if err != nil {
					log.Errorf("request error: %v", err)
					waitchan <- 1
				} else {
					log.Infof("recv reply message: %s", msg.Payload())
					waitchan <- 0
				}
			})

		res := <-waitchan
		if res == 1 {
			return
		}
		time.Sleep(1 * time.Second)
	}

	c.cli.Close()

	time.Sleep(10 * time.Second)
}
