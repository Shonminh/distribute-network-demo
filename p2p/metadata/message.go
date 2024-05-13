package metadata

import "fmt"

type Message struct {
	Type MsgType `json:"type"`
	Data []byte  `json:"data"`
}

func (m Message) String() string {
	return fmt.Sprintf("{type: %+v, data: %+v}", m.Type, string(m.Data))
}

type MsgType uint

const (
	Unknown        MsgType = 0 // 未知消息类型
	Ping           MsgType = 1 // ping消息，检查对端节点是否存活
	Pong           MsgType = 2 // pong消息，返回存活状态
	FindNodeReq    MsgType = 3 // 申请获取node请求消息
	FindNodeResult MsgType = 4 // 返回发现到的node列表结果
)

type DefaultMsg struct {
	ReqId  uint32
	NodeId string
}

type PingMsg struct {
	DefaultMsg
}

type PongMsg struct {
	DefaultMsg
}

type FindNodeMsg struct {
	ListenPort int
	DefaultMsg
}

type FindNodeResultMsg struct {
	Items []Item
	DefaultMsg
}
