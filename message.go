package ConnectWay

import "encoding/binary"

/*
	0-1: Magic number 0x55 0xaa
	2-5: Message Type
	6-9: Message length
	10-13: Message sequence
*/

type Message struct {
	header  [16]byte
	payload []byte
}

func CreateMessage(msgType uint32, msgLength uint32, seq uint32, msg []uint8) *Message {
	result := &Message{}
	// 组织头部
	result.header[0] = 0x55
	result.header[1] = 0xaa
	binary.LittleEndian.PutUint32(result.header[2:6], msgType)
	binary.LittleEndian.PutUint32(result.header[6:10], msgLength)
	binary.LittleEndian.PutUint32(result.header[10:14], seq)
	// 消息内容
	result.payload = msg
	return result
}

func (m *Message) Type() uint32 {
	msgtype := binary.LittleEndian.Uint32(m.header[2:6])
	return msgtype
}

func (m *Message) Length() uint32 {
	msglen := binary.LittleEndian.Uint32(m.header[6:10])
	return msglen
}

func (m *Message) Sequence() uint32 {
	seq := binary.LittleEndian.Uint32(m.header[10:14])
	return seq
}

func (m *Message) Payload() []byte {
	return m.payload
}
