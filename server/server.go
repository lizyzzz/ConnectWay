package main

import (
	cy "ConnectWay"
	"fmt"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
)

func InitSeeLog() {
	testConfig := `
	<seelog type="sync">
		<outputs formatid="common">
			<console/>
			<file path="./server.log"/>
		</outputs>
		<formats>
			<format id="common" format="[%Date(2006-01-02/15:04:05.000):%LEVEL:%File(%Line)] %Msg%n"/>
		</formats>
	</seelog>`
	logger, _ := log.LoggerFromConfigAsBytes([]byte(testConfig))
	log.ReplaceLogger(logger)
}

const (
	RecvMsgType uint32 = 0x55
	MsgType     uint32 = 0x65
	ReqMsgType  uint32 = 0x122
)

type MyServer struct {
	serv *cy.Server
}

func (s *MyServer) OnNewClient(stub *cy.ClientStub) {
	log.Infof("new client connected success, local: %s, remote: %s", stub.LocalAddr().String(), stub.RemoteAddr().String())
}

func (s *MyServer) OnStubMsg(msg *cy.Message, stub *cy.ClientStub) {
	log.Infof("recv msg: type: %d, payload: %s", msg.Type(), string(msg.Payload()))
	switch msg.Type() {
	case RecvMsgType:
		str := string(msg.Payload())
		index := strings.Index(str, ": ")
		num, _ := strconv.Atoi(str[index+2:])
		num *= -1
		repStr := fmt.Sprintf("hi num: %d", num)
		data := []byte(repStr)
		stub.Send(MsgType, uint32(len(data)), msg.Sequence(), data)
	case ReqMsgType:
		stub.Reply(ReqMsgType, msg.Length(), msg.Sequence(), msg.Payload())
	}
}

func (s *MyServer) OnErr(err error, stub *cy.ClientStub) {
	log.Errorf("stub err: %v", err)
}

func main() {
	InitSeeLog()
	mys := &MyServer{}
	mys.serv = cy.CreateServer("localhost:50051", mys.OnNewClient, mys.OnStubMsg, mys.OnErr)
	mys.serv.Start()
}
