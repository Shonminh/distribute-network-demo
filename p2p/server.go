package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"log"
	"net"
	"sync"
	"time"
)

var address = flag.String("address", "127.0.0.1:8080", "listen address, default 127.0.0.1:8080")

type Server struct {
	writerCh      chan *sendMessageGroup
	listenConn    *net.UDPConn
	wg            *sync.WaitGroup
	serverUdpAddr net.UDPAddr // 本地udp 信息
}

func init() {
	flag.Parse()
}

func NewServer() (*Server, error) {
	// 监听地址和端口
	addr := "127.0.0.1:8080"
	if address != nil {
		addr = *address
	}
	// 创建 UDP 地址结构
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	// 创建 UDP 连接
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &Server{writerCh: make(chan *sendMessageGroup, writerChannelBufferSize), listenConn: conn, wg: &sync.WaitGroup{}, serverUdpAddr: *udpAddr}, nil
}

func (server *Server) GetServerUdpAddr() net.UDPAddr {
	return server.serverUdpAddr
}

func (server *Server) Run() {
	defer server.close()

	server.wg.Add(3)
	// 异步写入udp数据
	go server.write()
	go server.listen()
	go server.discovery()
	server.wg.Wait()
}

func (server *Server) listen() {
	defer server.wg.Done()
	// 无限循环等待客户端请求
	for {
		// 读取数据
		buffer := make([]byte, msgBufferSize)
		n, clientAddr, err := server.listenConn.ReadFromUDP(buffer)
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

func (server *Server) close() {
	if server.listenConn == nil {
		return
	}
	_ = server.listenConn.Close()
}

var ErrUnknownMessage = errors.New("ErrUnknownMessage")
var ErrSendMsgTimeout = errors.New("ErrSendMsgTimeout")

func (server *Server) dispatcher(ctx context.Context, fromAddr *net.UDPAddr, rawData []byte) error {
	message := &metadata.Message{}
	_ = json.Unmarshal(rawData, message)
	switch message.Type {
	case metadata.Unknown:
		return ErrUnknownMessage
	case metadata.Ping:
		return server.ping(ctx, fromAddr, message)
	case metadata.Pong:
		return server.pong(ctx, fromAddr, message)
	case metadata.FindNodeReq:
		return server.findNode(ctx, fromAddr, message)
	case metadata.FindNodeResult:
		return server.findNodeResult(ctx, fromAddr, message)
	default:
		return nil
	}
}

const writerChannelBufferSize = 1024

func (server *Server) ping(ctx context.Context, fromAddr *net.UDPAddr, msg *metadata.Message) error {
	pingMsg := metadata.PingMsg{}
	_ = json.Unmarshal(msg.Data, &pingMsg)
	pongMsg := metadata.PongMsg{}
	pongMsg.ReqId = pingMsg.ReqId
	pongMsg.NodeId = getNodeId()
	marshal, _ := json.Marshal(pongMsg)
	resp := metadata.Message{Type: metadata.Pong, Data: marshal}
	data, _ := json.Marshal(resp)
	return server.sendWriter(ctx, &sendMessageGroup{sendData: data, toAddr: fromAddr})
}

func (server *Server) pong(ctx context.Context, fromAddr *net.UDPAddr, msg *metadata.Message) error {
	fmt.Printf("PONG, fromAddr=%+v, msg:%+v\n", fromAddr, msg)
	return nil
}

func (server *Server) findNode(ctx context.Context, fromAddr *net.UDPAddr, msg *metadata.Message) error {
	// TODO implement me
	return nil
}

func (server *Server) findNodeResult(ctx context.Context, fromAddr *net.UDPAddr, msg *metadata.Message) error {
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
	defer server.wg.Done()
	for msgGroup := range server.writerCh {
		_, err := server.listenConn.WriteToUDP(msgGroup.sendData, msgGroup.toAddr)
		if err != nil {
			log.Println("Error sending response: ", err)
		}
	}
}

func getNodeId() string {
	return "TODO implement me"
}
