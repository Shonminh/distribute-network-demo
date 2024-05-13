package p2p

import (
	"encoding/json"
	"fmt"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"github.com/google/uuid"
	"log"
	"net"
	"time"
)

func (server *Server) scheduleCheck() {
	defer server.wg.Done()
	for true {
		t := time.NewTimer(time.Second * 10)
		select {
		case <-t.C:
		}
		server.checkLiveness()
	}
}

func (server *Server) checkLiveness() {
	tables := server.tables
	tables.Mu.Lock()
	defer tables.Mu.Unlock()

	for i := range tables.Buckets {
		buckets := tables.Buckets[i]
		set := map[int]struct{}{}
		for j, item := range buckets.Items {
			if server.tables.LastSeen(*item) {
				continue
			}
			if !server.checkNodeLiveness(*item) {
				set[j] = struct{}{}
			} else {
				server.tables.SetSeen(*item)
			}
		}
		if len(set) == 0 {
			continue
		}
		var newItems []*metadata.Item
		for j := range buckets.Items {
			if _, ok := set[j]; !ok {
				newItems = append(newItems, buckets.Items[j])
			}
		}
		buckets.Items = newItems
	}
}

func (server *Server) checkNodeLiveness(address metadata.Item) bool {
	toUdpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", address.IP, address.Port))
	if err != nil {
		log.Printf("Error resolving Server address%+v, err=%+v", address, err)
		return false
	}
	fromUdpAddr := server.GetServerUdpAddr()
	if toUdpAddr.IP.String() == fromUdpAddr.IP.String() && fromUdpAddr.Port == toUdpAddr.Port {
		return false
	}
	conn, err := net.DialUDP("udp", nil, toUdpAddr)
	if err != nil {
		log.Printf("Error DialUDP address=%+v, err=%+v\n", address, err)
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	// 准备要发送的数据
	message, pingMsg := server.genPing()
	conn.Write(message)
	buffer := make([]byte, msgBufferSize)
	n, _, err := conn.ReadFrom(buffer)
	if err != nil {
		log.Printf("Error ReadFrom address%+v, err=%+v\n", address, err)
		return false
	}
	rawData := buffer[:n]
	msg := metadata.Message{}
	_ = json.Unmarshal(rawData, &msg)
	pongMsg := metadata.PongMsg{}
	_ = json.Unmarshal(msg.Data, &pongMsg)
	log.Printf("get pong response=%+v...\n", pongMsg)
	return msg.Type == metadata.Pong && pongMsg.ReqId == pingMsg.ReqId && pongMsg.NodeId == address.NodeId
}

func (server *Server) genPing() ([]byte, *metadata.PingMsg) {
	pingMsg := metadata.PingMsg{DefaultMsg: metadata.DefaultMsg{
		ReqId:  uuid.New().ID(),
		NodeId: server.nodeId,
	}}
	marshal, _ := json.Marshal(pingMsg)
	message := metadata.Message{
		Type: metadata.Ping,
		Data: marshal,
	}
	bytes, _ := json.Marshal(message)
	return bytes, &pingMsg
}

func (server *Server) scheduleFindNodes() {
	defer server.wg.Done()
	for true {
		t := time.NewTimer(time.Second * 10)
		select {
		case <-t.C:
		}
		server.findNodes()
	}
}

func (server *Server) scheduleSaveCfg() {
	defer server.wg.Done()
	for true {
		t := time.NewTimer(time.Second * 30)
		select {
		case <-t.C:
		}
		server.saveConfig()
	}
}

func (server *Server) findNodes() {
	// 找桶里面没有满的 找关联的最近的点的数据。
	tables := server.tables
	findClosestItem := tables.FindClosestItem(server.nodeId, 5)
	visit := map[metadata.Item]struct{}{}
	for _, node := range findClosestItem {
		visit[node] = struct{}{}
	}

	// bfs访问find remote node
	var q = findClosestItem
	for len(q) != 0 {
		item := q[0]
		q = q[1:]
		remoteNodes := server.findRemoteNode(item)
		for _, node := range remoteNodes {
			if _, ok := visit[node]; ok {
				continue
			}
			visit[node] = struct{}{}
			if tables.LastSeen(node) {
				continue
			}
			// 加入到本地hash表中
			tables.Load(node)
			q = append(q, node)
		}
	}
}

func (server *Server) findRemoteNode(address metadata.Item) []metadata.Item {
	toUdpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%v:%v", address.IP, address.Port))
	if err != nil {
		log.Printf("Error resolving Server address%+v, err=%+v", address, err)
		return nil
	}
	fromUdpAddr := server.GetServerUdpAddr()
	if toUdpAddr.IP.String() == fromUdpAddr.IP.String() && fromUdpAddr.Port == toUdpAddr.Port {
		return nil
	}
	conn, err := net.DialUDP("udp", nil, toUdpAddr)
	if err != nil {
		log.Printf("Error DialUDP address=%+v, err=%+v\n", address, err)
		return nil
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	message, _ := server.genFindNodeMsg()
	conn.Write(message)
	buffer := make([]byte, msgBufferSize)
	n, _, err := conn.ReadFrom(buffer)
	if err != nil {
		log.Printf("Error ReadFrom address%+v, err=%+v\n", address, err)
		return nil
	}
	rawData := buffer[:n]
	msg := metadata.Message{}
	_ = json.Unmarshal(rawData, &msg)
	nodeMsg := metadata.FindNodeResultMsg{}
	_ = json.Unmarshal(msg.Data, nodeMsg)
	log.Printf("get find node result msg=%+v...\n", nodeMsg)
	return nodeMsg.Items
}

func (server *Server) genFindNodeMsg() ([]byte, *metadata.FindNodeMsg) {
	findNodeMsg := metadata.FindNodeMsg{}
	findNodeMsg.ReqId = uuid.New().ID()
	findNodeMsg.NodeId = server.nodeId
	findNodeMsg.ListenPort = server.serverUdpAddr.Port // 本地监听的port
	marshal, _ := json.Marshal(findNodeMsg)
	message := metadata.Message{
		Type: metadata.FindNodeReq,
		Data: marshal,
	}
	bytes, _ := json.Marshal(message)
	return bytes, &findNodeMsg
}
