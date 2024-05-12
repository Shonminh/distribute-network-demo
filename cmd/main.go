package main

import (
	_ "github.com/Shonminh/distribute-network-demo/logger"
	"github.com/Shonminh/distribute-network-demo/p2p"
)

func main() {
	server, err := p2p.NewServer()
	if err != nil {
		panic(err)
	}
	server.Run()
}
