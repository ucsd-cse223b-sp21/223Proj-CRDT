package client

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"proj/network"
)

var (
	peer = flag.Int("peer", 0, "peer ID")
	numPeers = flag.Int("numPeers", 2, "number of peers")
)

// basic main for starting up a peer using config parsed from an argument
func main() {
	flag.Parse()
	configString := flag.Arg(1)

	config := network.Config{
		peer: 
		addrs: []string{
			// "",
			// ""
		}
	}
	err := json.Unmarshal([]byte(configString), &config)
	if err != nil {
		log.Panic("cannot unmarshal config from flag")
	}

	p := network.MakePeer(config)
	p.InitPeer()
	go p.Serve()

	// user input
}
