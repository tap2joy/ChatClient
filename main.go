package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tap2joy/ChatClient/utils"
	pb_common "github.com/tap2joy/Protocols/go/common"
	pb "github.com/tap2joy/Protocols/go/gateway"
	"google.golang.org/protobuf/proto"
)

var (
	IsLogin           = false
	IsStartGetChatLog = false
	IsChatLogReceived = false
	CurrentChannel    = uint32(1)
	NickName          = ""
)

func main() {
	gateAddress := utils.GetString("client", "gateway")
	conn, err := net.Dial("tcp", gateAddress)
	if err != nil {
		fmt.Printf("connect fail..., err = %v", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("connect server %s success\n", gateAddress)

	SendGetChannelListPacket(conn)

	go ReceiveHandle(conn)

	// 循环接收控制台输入
	for {
		if IsLogin {
			if !IsStartGetChatLog {
				SendGetLogPacket(conn)
			} else if IsChatLogReceived {
				cmdReader := bufio.NewReader(os.Stdin)
				cmdStr, _ := cmdReader.ReadString('\n')
				inputMsg := strings.Trim(cmdStr, "\r\n")

				if inputMsg == "quit" || inputMsg == "exit" {
					SendLogoutPacket(conn, NickName)
					break
				}

				if inputMsg == "" {
					continue
				}

				if strings.HasPrefix(inputMsg, "/switch") {
					// 切换频道
					params := strings.Split(inputMsg, " ")
					if len(params) >= 2 {
						targetChannelId, _ := strconv.Atoi(params[1])
						SendChangeChannelPacket(conn, NickName, uint32(targetChannelId))
						continue
					}
				}

				SendChatPacket(conn, NickName, inputMsg)
			}
		}
	}
}

// 发送登陆包
func SendLoginPacket(conn net.Conn, nick string, channelId uint32) {
	msg := &pb.CLogin{
		Name:    nick,
		Channel: channelId,
	}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	fmt.Printf("send user login packet %s channel %d\n", nick, channelId)
	SendPacket(conn, pb_common.Mid_C2G_USER_LOGIN, msgByte)
}

// 发送登出消息
func SendLogoutPacket(conn net.Conn, nick string) {
	fmt.Println("成功离开聊天室，再见")

	msg := &pb.CLogout{
		Name: nick,
	}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	SendPacket(conn, pb_common.Mid_C2G_USER_LOGOUT, msgByte)
}

func SendGetChannelListPacket(conn net.Conn) {
	msg := &pb.CGetChannelList{}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	SendPacket(conn, pb_common.Mid_C2G_GET_CHANNEL_LIST, msgByte)
}

func SendChangeChannelPacket(conn net.Conn, nick string, channelId uint32) {
	msg := &pb.CChangeChannel{
		Name:    nick,
		Channel: channelId,
	}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	SendPacket(conn, pb_common.Mid_C2G_CHANGE_CHANNEL, msgByte)
}

// 发送聊天消息包
func SendChatPacket(conn net.Conn, nick string, content string) {
	if nick == "" {
		fmt.Printf("nick can't be empty")
		return
	}
	if content == "" {
		fmt.Printf("content can't be empty")
		return
	}

	msg := &pb.CSend{
		SenderName: nick,
		Channel:    CurrentChannel,
		Content:    content,
	}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	SendPacket(conn, pb_common.Mid_C2G_SEND_MESSAGE, msgByte)
}

// 发送获取聊天记录的包
func SendGetLogPacket(conn net.Conn) {
	fmt.Println("开始拉取聊天记录")
	msg := &pb.CGetLog{
		Channel: CurrentChannel,
	}

	msgByte, err := proto.Marshal(msg)
	if err != nil {
		fmt.Printf("msg Marshal error %s\n", err.Error())
		return
	}

	IsStartGetChatLog = true
	SendPacket(conn, pb_common.Mid_C2G_GET_LOGS, msgByte)
}

func SendPacket(conn net.Conn, mid pb_common.Mid, msgByte []byte) {
	buf := &bytes.Buffer{}
	var head []byte
	head = make([]byte, 8)
	binary.BigEndian.PutUint32(head[0:4], uint32(bytes.Count(msgByte, nil)-1))
	binary.BigEndian.PutUint32(head[4:8], uint32(mid))
	buf.Write(head[:8])
	buf.Write(msgByte)

	conn.Write(buf.Bytes())
}

// 分包函数
func packetSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if !atEOF && len(data) >= 8 {
		var length int32
		// 读出 数据包中 实际数据 的长度
		binary.Read(bytes.NewReader(data[0:4]), binary.BigEndian, &length)
		packetLen := int(length) + 8
		dataLen := len(data)
		//fmt.Printf("packetLen = %d, dataLen = %d\n", packetLen, dataLen)
		if packetLen <= dataLen {
			return packetLen, data[:packetLen], nil
		}
	}
	return
}

// 客户端接收消息处理
func ReceiveHandle(conn net.Conn) {
	for {
		readData := make([]byte, 8192)
		readLen, err := conn.Read(readData)

		if err != nil {
			if err == io.EOF {
				continue
			}

			fmt.Println("conn closed")
			os.Exit(0)
		}

		if readLen == 0 {
			fmt.Println("readLen == 0")
			continue
		}

		// 处理tcp粘包
		buf := bytes.NewBuffer(readData[0:readLen])
		scanner := bufio.NewScanner(buf)
		scanner.Split(packetSplitFunc)

		for scanner.Scan() {
			packetBytes := scanner.Bytes()
			HandleServerPacket(conn, packetBytes)
		}

		if err := scanner.Err(); err != nil {
			fmt.Println("无效数据包")
			continue
		}
	}
}

func HandleServerPacket(conn net.Conn, dataBytes []byte) {
	// 读取包头
	length := int32(0)
	var mid pb_common.Mid
	binary.Read(bytes.NewReader(dataBytes[0:4]), binary.BigEndian, &length)
	binary.Read(bytes.NewReader(dataBytes[4:8]), binary.BigEndian, &mid)

	if mid == pb_common.Mid_INVALID_MID || mid > 9999 {
		//fmt.Printf("invalid mid %d \n", mid)
		return
	}

	// 对接收到的数据进行解码
	switch mid {
	case pb_common.Mid_G2C_USER_LOGIN:
		msg := &pb.SLogin{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal login error:", err)
		}

		HandleLoginResp(msg)
		break
	case pb_common.Mid_G2C_USER_LOGOUT:
		msg := &pb.SLogout{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal user logout error:", err)
		}

		HandleLogoutResp(msg)
		break
	case pb_common.Mid_G2C_SEND_MESSAGE:
		msg := &pb.SSend{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal send error:", err)
		}

		HandleSendResp(msg)
		break
	case pb_common.Mid_G2C_GET_LOGS:
		msg := &pb.SGetLog{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal chat logs error:", err)
		}

		HandleGetLogResp(msg)
		break
	case pb_common.Mid_G2C_PUSH_MESSAGE:
		msg := &pb.SPushMessage{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal push message error:", err)
		}

		HandlePushMessage(msg)
		break
	case pb_common.Mid_G2C_CHANGE_CHANNEL:
		msg := &pb.SChangeChannel{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal change channel error:", err)
		}

		HandleChangeChannel(msg)
		break
	case pb_common.Mid_G2C_GET_CHANNEL_LIST:
		msg := &pb.SGetChannelList{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal get channel list error:", err)
		}

		HandleGetChannelListResp(conn, msg)
		break
	case pb_common.Mid_G2C_ERROR_MESSAGE:
		msg := &pb_common.SErrorMessage{}
		err := proto.Unmarshal(dataBytes[8:length+8], msg)
		if err != nil {
			fmt.Println("Unmarshal error message error:", err)
		}

		HandleErrorMessage(msg)
		break
	default:
		fmt.Printf("unknown mid %d\n", mid)
	}
}

// 处理登陆回复消息
func HandleLoginResp(msg *pb.SLogin) {
	if msg == nil {
		return
	}

	CurrentChannel = msg.Channel

	fmt.Println("-------------------------------------")
	fmt.Printf("**** 欢迎: %s 当前聊天室：%d ****\n", msg.Name, msg.Channel)
	fmt.Println("-------------------------------------")
	IsLogin = true
}

// 处理登出回复
func HandleLogoutResp(msg *pb.SLogout) {
	if msg == nil {
		return
	}

	fmt.Println("-------------------------------------")
	fmt.Printf("**** 您已成功离开聊天室，再见 %s ****\n", msg.Name)
	fmt.Println("-------------------------------------")
	IsLogin = false
	os.Exit(0)
}

// 处理聊天消息回复
func HandleSendResp(msg *pb.SSend) {
	if msg == nil {
		return
	}

	if msg.Result != "" {
		timeStr := formatTime(time.Now().Unix())
		fmt.Printf("系统消息 %s\n", timeStr)
		fmt.Printf("    %s\n", msg.Result)
	}
}

// 处理聊天记录回复
func HandleGetLogResp(msg *pb.SGetLog) {
	if msg == nil {
		return
	}

	IsChatLogReceived = true
	// 倒序显示
	for i := len(msg.Logs) - 1; i >= 0; i-- {
		v := msg.Logs[i]
		timeStr := formatTime(int64(v.Timestamp))
		fmt.Printf("%s %s\n", v.SenderName, timeStr)
		fmt.Printf("    %s\n", v.Content)
	}
}

// 处理推送消息
func HandlePushMessage(msg *pb.SPushMessage) {
	if msg == nil {
		return
	}

	timeStr := formatTime(int64(msg.Timestamp))
	fmt.Printf("%s %s\n", msg.SenderName, timeStr)
	fmt.Printf("    %s\n", msg.Content)
}

// 处理切换频道
func HandleChangeChannel(msg *pb.SChangeChannel) {
	if msg == nil {
		return
	}

	CurrentChannel = msg.Channel
	fmt.Println("-------------------------------------")
	fmt.Printf("当前聊天室id为 %d\n", CurrentChannel)
	fmt.Println("-------------------------------------")

	// 聊天记录倒序显示
	for i := len(msg.Logs) - 1; i >= 0; i-- {
		v := msg.Logs[i]
		timeStr := formatTime(int64(v.Timestamp))
		fmt.Printf("%s %s\n", v.SenderName, timeStr)
		fmt.Printf("    %s\n", v.Content)
	}
}

// 处理获取频道列表
func HandleGetChannelListResp(conn net.Conn, msg *pb.SGetChannelList) {
	if msg == nil {
		return
	}

	fmt.Println("当前可用的聊天室:")
	for _, v := range msg.List {
		fmt.Printf("    %d : %s\n", v.Id, v.Desc)
	}

	var channelStr string
	fmt.Println("请输入您想要进入的聊天室id：")
	_, err := fmt.Scanln(&channelStr)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	channelId, err := strconv.Atoi(channelStr)
	if err != nil {
		fmt.Printf("channel err: %v\n", err)
	}

	fmt.Println("请输入您在聊天室中要显示的昵称：")
	cmdReader := bufio.NewReader(os.Stdin)
	cmdStr, _ := cmdReader.ReadString('\n')
	NickName = strings.Trim(cmdStr, "\r\n")

	// 发送登陆消息
	go SendLoginPacket(conn, NickName, uint32(channelId))
}

// 显示错误消息
func HandleErrorMessage(msg *pb_common.SErrorMessage) {
	if msg == nil {
		return
	}

	timeStr := formatTime(time.Now().Unix())
	fmt.Printf("系统消息 %s\n", timeStr)
	fmt.Printf("    error code: %d, msg: %s \n", msg.Code, msg.Msg)
}

// 格式化消息时间
func formatTime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}
