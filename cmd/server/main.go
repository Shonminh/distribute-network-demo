package main

import (
	_ "github.com/Shonminh/distribute-network-demo/logger"
	"github.com/Shonminh/distribute-network-demo/p2p"
)

func main() {
	p2p.Listen()
}
