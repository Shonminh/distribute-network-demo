package main

import (
	_ "github.com/Shonminh/distribute-network-demo/logger"
	"github.com/Shonminh/distribute-network-demo/p2p"
	"time"
)

func main() {
	go p2p.Discovery()
	go p2p.CheckLiveness()
	// select {}
	time.Sleep(time.Second / 2)
}
