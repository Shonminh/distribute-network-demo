package p2p

import (
	"encoding/json"
	"fmt"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"net"
	"time"
)

func Discovery() {

	// 服务器地址和端口
	serverAddr := "127.0.0.1:8080"

	// 解析服务器地址
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println("Error resolving Server address:", err)
		return
	}

	// 创建 UDP 连接
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println("Error connecting to Server:", err)
		return
	}
	defer conn.Close()

	// 准备要发送的数据
	message := genPing()
	// 发送数据
	_, err = conn.Write(message)
	if err != nil {
		fmt.Println("Error sending data:", err)
		return
	}

	// 接收响应
	buffer := make([]byte, msgBufferSize)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	fmt.Println("Response from Server:", string(buffer[:n]))
}

func genPing() []byte {
	pingMsg := metadata.PingMsg{DefaultMsg: metadata.DefaultMsg{
		ReqId:  uint(time.Now().Unix()),
		NodeId: "本地",
	}}
	marshal, _ := json.Marshal(pingMsg)
	message := metadata.Message{
		Type: metadata.Ping,
		Data: marshal,
	}
	bytes, _ := json.Marshal(message)
	return bytes
}
