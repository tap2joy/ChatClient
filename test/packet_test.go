package test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"

	pb_common "github.com/tap2joy/Protocols/go/common"
	pb "github.com/tap2joy/Protocols/go/gateway"
	"google.golang.org/protobuf/proto"
)

func TestPacketLength(t *testing.T) {
	fmt.Println("====== packet lenght test begin")

	msg := &pb.CSend{}
	msg.SenderName = "aaa"
	msg.Channel = 1
	msg.Content = "恭喜发财"

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	mid := pb_common.Mid_C2G_SEND_MESSAGE
	dataLen := uint32(bytes.Count(msgByte, nil) - 1)
	fmt.Println("dataLen = ", dataLen)

	dataLen1 := uint32(len(msgByte))
	fmt.Println("dataLen1 = ", dataLen1)

	// write
	buf := &bytes.Buffer{}
	var head []byte
	head = make([]byte, 8)
	binary.BigEndian.PutUint32(head[0:4], dataLen1)
	binary.BigEndian.PutUint32(head[4:8], uint32(mid))
	buf.Write(head[:8])
	buf.Write(msgByte)

	bufLen := len(buf.Bytes())
	fmt.Println("bufLen = ", bufLen)

	// read
	data := buf.Bytes()
	var readPacketLength uint32
	binary.Read(bytes.NewReader(data[0:4]), binary.BigEndian, &readPacketLength)
	fmt.Println("readPacketLength = ", readPacketLength)

	var readMid uint32
	binary.Read(bytes.NewReader(data[0:4]), binary.BigEndian, &readMid)
	fmt.Println("readMid = ", readMid)

	readMsg := &pb.CSend{}
	err = proto.Unmarshal(data[8:readPacketLength+8], readMsg)
	if err != nil {
		fmt.Println("Unmarshal push message error:", err)
	}
	fmt.Println("readMsg = ", readMsg)

}
