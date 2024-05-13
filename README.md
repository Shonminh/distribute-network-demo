# distribute-network-demo
distribute network demo

# 项目介绍
实现了一个简易版本的分布式hash表（DHT），具体参考这篇文章：[kademlia](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf)
本项目目前只做了节点发现的功能。
大致实现方式：
每个节点存在一个逻辑的node id，使用sha256进行hash，长度为32 * 8 = 256 bit。
每个节点在内存里面维护一个路由表Table，表中桶Bucket的固定长度K = 20，table中桶的下标与其他节点的node id和当前节点的node id的逻辑距离有关。
逻辑距离使用xor进行异或计算。

**逻辑距离计算逻辑**
```go
// a 和b都是hex 格式的64长度
var m = map[byte]uint8{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9, 'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14, 'f': 15}

// Distance 找前缀树的长度，算距离
func Distance(a, b string) int {
	count := 0
	for i := 0; i < 64; i += 2 {
		x := m[a[i]]*16 + m[a[i+1]]
		y := m[b[i]]*16 + m[b[i+1]]
		z := x ^ y

		if z == 0 { //
			count += 8
		} else {
			count += bits.LeadingZeros8(z)
			break
		}
	}
	return sha256Len*8 - count
}
```
连续的前缀bit位相同的位数越长，那么逻辑距离越近，表示两个节点越是相邻的。距离的范围[0, 256]。

**桶下标位置定位**
桶的个数为20，将距离为[0, 256]的节点列表映射到[0, 19]的区间，逻辑距离在`lowDistanceSize`内的映射到下标0的桶中，其余[`lowDistanceSize`, 256]的节点映射到
[0, 19]中。其中256 - `lowDistanceSize` = 20。这样减少桶的数据分布不均的问题。

**节点发现**
新的节点初始化会使用`InitialNodeAddress`中的node id做引导，
然后递归的进行FIND_NODE操作，从引导节点获取离当前节点较近的节点列表，然后针对返回的列表进行递归查询，并加入到路由表中。
定时保存路由表的数据到文件中。

**节点监听**
监听PING操作和FIND_NODE操作，针对PING操作，直接返回PONG消息即可，针对FIND_NODE操作，将请求的节点信息尝试加入到本地路由表，并在本地路由表中查找指定数量的
closest节点数量。

**节点探活**
针对路由表中存的节点，维护上次探活的时间，如果最近刚探活过则忽略，没有的话则发送PING操作，向目标节点发送UDP请求。如果正常则更新
探活时间，如果不正常则从路由表进行剔除。

# 如何使用
1. 进入项目目录
> cd distribute-network-demo
2. 启动引导节点
> go run ./cmd/main.go -address=127.0.0.1:8080
> 
> go run ./cmd/main.go -address=127.0.0.1:8081 -configFile=config8081.json
> 
> go run ./cmd/main.go -address=127.0.0.1:8082 -configFile=config8082.json


3. 加入新节点

比如 新加入一个UDP监听的端口号为8083的进程，且对应的node id和路由表的配置存放的文件目录为`config8083.json`
> go run ./cmd/main.go -address=127.0.0.1:8083 -configFile=config8083.json

4. 退出节点
> 直接关掉进程


# 待完善
- 节点剔除，可以增加一个额外的等待剔除的节点列表，如果探活失败的次数过多则从Bucket中踢掉。
- 探活本地路由表中的元素的机制可以更多一些，比如随机探活。