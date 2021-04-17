package test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"testing"

	pb_common "github.com/tap2joy/Protocols/go/common"
	pb "github.com/tap2joy/Protocols/go/gateway"
	"google.golang.org/protobuf/proto"
)

func ReceiveHandle(conn net.Conn) {
	for {
		readData := make([]byte, 64535)
		readLen, err := conn.Read(readData)

		if err != nil {
			if err == io.EOF {
				fmt.Println("conn closed")
				break
			}
			continue
		}

		if readLen == 0 {
			fmt.Println("readLen == 0")
			continue
		}

		length := int32(0)
		var mid pb_common.Mid
		binary.Read(bytes.NewReader(readData[0:4]), binary.BigEndian, &length)
		binary.Read(bytes.NewReader(readData[4:8]), binary.BigEndian, &mid)

		msg := &pb.SGetChannelList{}
		err = proto.Unmarshal(readData[8:], msg)
		if err != nil {
			log.Fatal("Unmarshal error:", err)
		}

		fmt.Printf("%v\n", msg)
	}
}

func TestGetChannelList(t *testing.T) {
	gateAddress := "127.0.0.1:9108"
	conn, err := net.Dial("tcp", gateAddress)
	if err != nil {
		fmt.Printf("connect fail..., err = %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	go ReceiveHandle(conn)

	msg := &pb.CGetChannelList{}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	buf := &bytes.Buffer{}
	var head []byte
	head = make([]byte, 8)
	binary.BigEndian.PutUint32(head[0:4], uint32(bytes.Count(msgByte, nil)-1))
	binary.BigEndian.PutUint32(head[4:8], uint32(pb_common.Mid_C2G_GET_CHANNEL_LIST))
	buf.Write(head[:8])
	buf.Write(msgByte)

	conn.Write(buf.Bytes())
}
