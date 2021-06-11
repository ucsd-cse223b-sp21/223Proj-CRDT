package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"proj/network"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	peer     = flag.Int("peer", 0, "peer ID")
	numPeers = flag.Int("numPeers", 2, "number of peers (max 8)")
	gui      = flag.Bool("gui", false, "cli or gui")
	upgrader = websocket.Upgrader{}
)

// basic main for starting up a peer using config parsed from an argument
func main() {
	flag.Parse()

	config := network.Config{
		Peer: *peer,
		Addrs: []string{
			"localhost:3001",
			"localhost:3002",
			"localhost:3003",
			"localhost:3004",
			"localhost:3005",
			"localhost:3006",
			"localhost:3007",
			"localhost:3008",
		}[:*numPeers],
	}

	p := network.MakePeer(config)
	if *gui {
		single := sync.Mutex{} // force single gui per peer to avoid concurrent writes to the same peer

		handler := func(w http.ResponseWriter, r *http.Request) {
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Print("websocket handler failed on upgrade")
				return
			}

			single.Lock()
			p.Rga.Doc.AddFront(c)
			for {
				_, buf, err := c.ReadMessage()
				if err != nil {
					break
				}
				args := fields(string(buf))
				if len(args) > 0 {
					if runCmd(p, args) {
						break
					}
				}
			}
			c.Close()
			single.Unlock()
		}
		p.InitPeer(handler)

		// wait forever
		var nilC chan bool
		nilC <- true
	}

	p.InitPeer(nil)
	scanner := bufio.NewScanner(os.Stdin)

	// user input
	for scanner.Scan() {
		fmt.Print("> ")
		line := scanner.Text()
		args := fields(line)
		if len(args) > 0 {
			if runCmd(p, args) {
				break
			}
		}
		fmt.Println()
	}
}

func fields(s string) []string {
	return strings.Fields(s)
}

func noError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

func logError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}

func runCmd(p *network.Peer, args []string) bool {
	cmd := args[0]

	log.Printf("rga pointer in cmd is %p", p.Rga)

	switch cmd {
	case "append":
		i, err := strconv.Atoi(args[1])
		logError(err)
		logError(p.Rga.Doc.Append(i, args[2][0]))
	case "remove":
		i, err := strconv.Atoi(args[1])
		logError(err)
		logError(p.Rga.Doc.Remove(i))
	case "view":
		fmt.Println("=== | VIEW  | ===")
		fmt.Println(p.Rga.Doc.View())
		fmt.Println("=== | END  | ===")
	case "update":
		p.Rga.Doc.ComputeView()
	case "connect":
		p.Connect()
	case "disconnect":
		p.Disconnect()
	default:
		logError(fmt.Errorf("bad command, try \"help\"."))
	}
	return false
}
