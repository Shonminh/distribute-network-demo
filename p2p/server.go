package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"github.com/Shonminh/distribute-network-demo/p2p/crypto"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var address = flag.String("address", "127.0.0.1:8080", "listen address, default 127.0.0.1:8080")
var configFile = flag.String("configFile", "config.json", "config file")

type Server struct {
	writerCh      chan *sendMessageGroup
	listenConn    *net.UDPConn
	wg            *sync.WaitGroup
	serverUdpAddr net.UDPAddr // 本地udp 信息
	nodeId        string      // 从本地文件中找，没有则生成。
	cfg           config
	cfgMutex      sync.Mutex
	once          *sync.Once
	tables        *metadata.Table // dht路由表
}

type config struct {
	NodeId          string          `json:"NodeId"`
	RoutingNodeList []metadata.Item `json:"RoutingNodeList"`
}

func init() {
	flag.Parse()
}

func (server *Server) loadConfig() {
	server.once.Do(func() {
		server.cfgMutex.Lock()
		defer server.cfgMutex.Unlock()
		file, err := os.OpenFile(*configFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
		defer file.Close()
		if err != nil {
			panic(err)
		}
		all, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		cfg := config{}
		_ = json.Unmarshal(all, &cfg)
		server.cfg = cfg
		server.nodeId = cfg.NodeId
	})
}

func (server *Server) saveConfig() {
	server.cfgMutex.Lock()
	defer server.cfgMutex.Unlock()
	server.cfg.RoutingNodeList = server.tables.GetAllItems()
	marshal, _ := json.Marshal(server.cfg)
	_ = os.WriteFile(*configFile, marshal, 0744)
}

// 启动的时候把从引导节点的配置和之前存到配置文件发现的node load到table中
func (server *Server) loadTable() {
	tables := server.tables
	for _, v := range server.cfg.RoutingNodeList {
		tables.Load(v)
	}
	// 引导节点
	for _, v := range metadata.InitialNodeAddress {
		tables.Load(v)
	}
}

func (server *Server) saveNodeIdIfNeed() {
	if server.nodeId == "" {
		server.cfgMutex.Lock()
		defer server.cfgMutex.Unlock()
		nodeId := crypto.GenNodeId()
		server.nodeId = nodeId
		server.cfg.NodeId = nodeId
		marshal, _ := json.Marshal(server.cfg)
		err := os.WriteFile(*configFile, marshal, 0744)
		if err != nil {
			panic(err)
		}
	}
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
	srv := &Server{writerCh: make(chan *sendMessageGroup, writerChannelBufferSize), listenConn: conn, wg: &sync.WaitGroup{},
		serverUdpAddr: *udpAddr, once: &sync.Once{}}
	// 加载配置
	srv.loadConfig()
	srv.saveNodeIdIfNeed()
	tables := metadata.NewTable(srv.nodeId)
	srv.tables = tables
	srv.loadTable()
	return srv, nil
}

func (server *Server) GetServerUdpAddr() net.UDPAddr {
	return server.serverUdpAddr
}

func (server *Server) Run() {
	defer server.close()

	server.wg.Add(5)
	// 异步发送udp响应
	go server.write()
	// 监听udp请求
	go server.listen()
	// 本地路由表保存的节点状态检查
	go server.scheduleCheck()
	// 节点发现
	go server.scheduleFindNodes()
	// 保存配置
	go server.scheduleSaveCfg()
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
	log.Printf("Received message from %s: %s\n", fromAddr.String(), message)
	switch message.Type {
	case metadata.Unknown:
		return ErrUnknownMessage
	case metadata.Ping:
		return server.ping(ctx, fromAddr, message)
	case metadata.FindNodeReq:
		return server.findNode(ctx, fromAddr, message)
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
	pongMsg.NodeId = server.nodeId
	marshal, _ := json.Marshal(pongMsg)
	resp := metadata.Message{Type: metadata.Pong, Data: marshal}
	data, _ := json.Marshal(resp)
	return server.sendWriter(ctx, &sendMessageGroup{sendData: data, toAddr: fromAddr})
}

// 查找本地路由表中的数据
func (server *Server) findNode(ctx context.Context, fromAddr *net.UDPAddr, msg *metadata.Message) error {

	findNodeMsg := metadata.FindNodeMsg{}
	_ = json.Unmarshal(msg.Data, &findNodeMsg)
	// 将请求的node id加入到本地路由表。
	server.tables.AddRequestNodeId(findNodeMsg.NodeId, fromAddr.IP.String(), findNodeMsg.ListenPort)

	// 搜索最近的k bucket中的元素返回。
	closestItems := server.tables.FindClosestItem(findNodeMsg.NodeId, 5)
	findNodeResultMsg := metadata.FindNodeResultMsg{Items: closestItems}
	findNodeResultMsg.ReqId = findNodeMsg.ReqId
	findNodeResultMsg.NodeId = server.nodeId
	marshal, _ := json.Marshal(findNodeResultMsg)
	resp := metadata.Message{
		Type: metadata.FindNodeResult,
		Data: marshal,
	}
	data, _ := json.Marshal(resp)
	return server.sendWriter(ctx, &sendMessageGroup{sendData: data, toAddr: fromAddr})
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
