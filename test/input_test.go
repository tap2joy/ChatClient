package test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestScanInput(t *testing.T) {
	var nickName string
	fmt.Println("请输入您在聊天室中要显示的昵称：")
	fmt.Scan(&nickName)
	fmt.Printf("your input: %s\n", nickName)
}

func TestBuffIO(t *testing.T) {
	cmdReader := bufio.NewReader(os.Stdin)
	cmdStr, _ := cmdReader.ReadString('\n')
	cmdStr = strings.Trim(cmdStr, "\r\n")
	fmt.Printf("your input: %s\n", cmdStr)
}
