package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"log"
	"net"
	"time"
)

func Listen() {
	// 监听地址和端口
	addr := ":8080"

	// 创建 UDP 地址结构
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	// 创建 UDP 连接
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer conn.Close()

	fmt.Println("UDP Server is running on", addr)
	server := NewServer(conn)

	// 异步写入udp数据
	go server.write()

	// 无限循环等待客户端请求
	for {
		// 读取数据
		buffer := make([]byte, msgBufferSize)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error reading from UDP connection:", err)
			continue
		}
		rawData := buffer[:n]

		log.Printf("Received message from %s: %s\n", clientAddr.String(), string(rawData))
		if err = server.dispatcher(context.Background(), clientAddr, rawData); err != nil {
			log.Printf("failed to process message[%+v]...\n", string(rawData))
		}
	}
}

type Server struct {
	writerCh chan *sendMessageGroup // make(chan metadata.Message, writerChannelBufferSize)
	conn     *net.UDPConn
}

func NewServer(conn *net.UDPConn) *Server {
	return &Server{writerCh: make(chan *sendMessageGroup, writerChannelBufferSize), conn: conn}
}

var ErrUnknownMessage = errors.New("ErrUnknownMessage")
var ErrSendMsgTimeout = errors.New("ErrSendMsgTimeout")

func (server *Server) dispatcher(ctx context.Context, clientAddr *net.UDPAddr, rawData []byte) error {
	message := metadata.Message{}
	_ = json.Unmarshal(rawData, &message)
	switch message.Type {
	case metadata.Unknown:
		return ErrUnknownMessage
	case metadata.Ping:
		return server.ping(ctx, clientAddr, &message)
	case metadata.FindNodeReq:
		return server.findNode(ctx, clientAddr, &message)
	}
	return nil
}

const writerChannelBufferSize = 1024

func (server *Server) ping(ctx context.Context, clientAddr *net.UDPAddr, msg *metadata.Message) error {
	pingMsg := metadata.PingMsg{}
	_ = json.Unmarshal(msg.Data, &pingMsg)
	pongMsg := metadata.PongMsg{}
	pongMsg.ReqId = pingMsg.ReqId
	pongMsg.NodeId = getNodeId()
	marshal, _ := json.Marshal(pongMsg)
	resp := metadata.Message{Type: metadata.Pong, Data: marshal}
	data, _ := json.Marshal(resp)
	return server.sendWriter(ctx, &sendMessageGroup{sendData: data, toAddr: clientAddr})
}

func (server *Server) findNode(ctx context.Context, clientAddr *net.UDPAddr, msg *metadata.Message) error {
	// TODO implement me
	return nil
}

type sendMessageGroup struct {
	toAddr   *net.UDPAddr
	sendData []byte
}

func (server *Server) sendWriter(ctx context.Context, msgGroup *sendMessageGroup) error {
	timer := time.NewTimer(time.Second)
	select {
	case server.writerCh <- msgGroup:
		return nil
	case <-timer.C:
		return ErrSendMsgTimeout
	case <-ctx.Done():
		return ErrSendMsgTimeout
	}
}

func (server *Server) write() {
	for msgGroup := range server.writerCh {
		_, err := server.conn.WriteToUDP(msgGroup.sendData, msgGroup.toAddr)
		if err != nil {
			log.Println("Error sending response: ", err)
		}
	}
}

func getNodeId() string {
	return "TODO implement me"
}
