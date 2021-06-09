package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"proj/network"
	"strconv"
	"strings"
)

var (
	peer     = flag.Int("peer", 0, "peer ID")
	numPeers = flag.Int("numPeers", 2, "number of peers (max 8)")
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
	log.Printf("rga pointer in cmd is %p", p.Rga)
	p.InitPeer()
	log.Printf("rga pointer in cmd is %p", p.Rga)

	scanner := bufio.NewScanner(os.Stdin)

	// user input
	for {
		fmt.Print("> ")
		for scanner.Scan() {
			line := scanner.Text()
			args := fields(line)
			if len(args) > 0 {
				if runCmd(p, args) {
					break
				}
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
	p.Rga.B()

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
		p.Rga.Doc.UpdateView()
	case "connect":
		p.Connect()
	case "disconnect":
		p.Disconnect()
	default:
		logError(fmt.Errorf("bad command, try \"help\"."))
	}
	return false
}
