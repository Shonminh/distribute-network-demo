package metadata

import (
	"math/bits"
)

var InitialNodeAddress = []Item{
	{"127.0.0.1", 8080, "dadda59714ace634f7d79d7996a40185d2d6d1ee4cb5a88999075a51536c241d"},
	{"127.0.0.1", 8081, "dadda59714ace634f7d79d7996a40185d2d6d1ee4cb5a88999075a51536c241f"},
	{"127.0.0.1", 8082, "dadda59714ace634f7d79d7996a40185d2d6d1ee4cb5a88999075a51536c241e"},
}

type Bucket struct {
	Items []*Item
}

type Item struct {
	IP     string
	Port   int
	NodeId string
}

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
