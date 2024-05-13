package metadata

import (
	"sync"
	"time"
)

const (
	K                 = 20 // Buckets最大数量
	maxBucketSize     = 20 // 每个桶的最大容量
	sha256Len         = 32
	lowDistanceSize   = sha256Len*8 - K
	lastCheckDuration = 3600 // 3600秒
)

type Table struct {
	Mu           *sync.Mutex
	NodeId       string
	ItemSet      map[Item]struct{} // 去重用的
	LastCheckMap map[Item]int64
	Buckets      [K]*Bucket
}

func NewTable(nodeId string) *Table {
	buckets := [K]*Bucket{}
	for i := range buckets {
		buckets[i] = &Bucket{}
	}
	res := &Table{Mu: &sync.Mutex{}, NodeId: nodeId, Buckets: buckets, ItemSet: map[Item]struct{}{}, LastCheckMap: map[Item]int64{}}
	return res
}

func (t *Table) SetSeen(item Item) {
	t.LastCheckMap[item] = time.Now().Unix()
}

func (t *Table) LastSeen(item Item) bool {
	lastSeen, ok := t.LastCheckMap[item]
	if !ok {
		return false
	}
	// 一小时之内没有ping过
	return time.Now().Unix()-lastSeen < lastCheckDuration
}

func (t *Table) GetBucket(fromNodeId string) *Bucket {
	return t.Buckets[t.GetBucketIdx(fromNodeId)]
}

func (t *Table) GetBucketIdx(fromNodeId string) int {
	distance := Distance(t.NodeId, fromNodeId)
	if distance <= lowDistanceSize {
		return 0
	}
	return distance - lowDistanceSize - 1 // [0, 19]
}

func (t *Table) AddRequestNodeId(fromNodeId string, ip string, port int) {
	item := Item{IP: ip, Port: port, NodeId: fromNodeId}
	if _, ok := t.ItemSet[item]; ok {
		return
	}
	t.Mu.Lock()
	defer t.Mu.Unlock()
	bucket := t.GetBucket(fromNodeId)
	if len(bucket.Items) >= maxBucketSize {
		return
	}
	bucket.Items = append(bucket.Items, &item)
}

// FindClosestItem 查找本地表中的最近的节点
func (t *Table) FindClosestItem(fromNodeId string, total int) []Item {
	idx := t.GetBucketIdx(fromNodeId)
	var res []Item
	for i := 0; i < len(t.Buckets[idx].Items); i++ {
		res = append(res, *t.Buckets[idx].Items[i])
	}
	if len(res) >= total {
		return res[:total]
	}

	for i := idx - 1; i >= 0 && len(res) < total; i-- {
		for j := 0; j < len(t.Buckets[i].Items) && len(res) < total; j++ {
			res = append(res, *t.Buckets[i].Items[j])
		}
	}
	if len(res) >= total {
		return res[:total]
	}
	for i := idx + 1; i < len(t.Buckets) && len(res) < total; i++ {
		for j := 0; j < len(t.Buckets[i].Items) && len(res) < total; j++ {
			res = append(res, *t.Buckets[i].Items[j])
		}
	}
	if len(res) >= total {
		return res[:total]
	}
	return res
}

func (t *Table) Load(item Item) {
	t.Mu.Lock()
	defer t.Mu.Unlock()
	if _, ok := t.ItemSet[item]; ok {
		return
	}
	// 自身节点不加。
	if t.NodeId == item.NodeId {
		return
	}
	bucket := t.GetBucket(item.NodeId)
	if len(bucket.Items) >= maxBucketSize {
		return
	}
	bucket.Items = append(bucket.Items, &item)
	t.ItemSet[item] = struct{}{}
}

func (t *Table) GetAllItems() []Item {
	t.Mu.Lock()
	defer t.Mu.Unlock()
	var res []Item
	for _, bucket := range t.Buckets {
		for _, item := range bucket.Items {
			if item == nil {
				continue
			}
			res = append(res, *item)
		}
	}
	return res
}
