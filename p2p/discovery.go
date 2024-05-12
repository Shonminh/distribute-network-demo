package p2p

import (
	"encoding/json"
	"github.com/Shonminh/distribute-network-demo/p2p/metadata"
	"log"
	"net"
	"time"
)

func (server *Server) checkLiveness(address string) {
	toUdpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Printf("Error resolving Server address%+v, err=%+v", address, err)
		return
	}
	fromUdpAddr := server.GetServerUdpAddr()
	if toUdpAddr.IP.String() == fromUdpAddr.IP.String() && fromUdpAddr.Port == toUdpAddr.Port {
		return
	}
	conn, err := net.DialUDP("udp", nil, toUdpAddr)
	if err != nil {
		log.Printf("Error DialUDP address=%+v, err=%+v\n", address, err)
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	// 准备要发送的数据
	message := genPing()
	conn.Write(message)
	buffer := make([]byte, msgBufferSize)
	n, _, err := conn.ReadFrom(buffer)
	if err != nil {
		log.Printf("Error ReadFrom address%+v, err=%+v\n", address, err)
	}
	rawData := buffer[:n]
	log.Println("rawData: ", string(rawData))
}

func (server *Server) discovery() {
	defer server.wg.Done()
	for true {
		t := time.NewTimer(time.Second * 10)
		select {
		case <-t.C:
		}
		for _, address := range metadata.InitialNodeAddress {
			server.checkLiveness(address)
		}
	}
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
