package network

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"

	"crdt"

	"github.com/gorilla/websocket"
)

type Config struct {
	peer  int
	addrs []string
}

type Message struct {
	e  crdt.Elem
	vc crdt.VecClock
}

var (
	config = flag.String("addr", "", "address of server")
)

var peer int
var addrs []string
var conns map[*websocket.Conn]bool
var upgrader = websocket.Upgrader{}
var vc crdt.VecClock
var rga crdt.RGA

func initialize(c Config) {
	peer = c.peer
	addrs = c.addrs

	conns = make(map[*websocket.Conn]bool)
	vc = crdt.VecClock{peer: peer, vc: make([]uint64, len(addrs))}
	rga = crdt.newRGA(peer, len(addrs))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("websocket handler failed on upgrade")
		return
	}

	conns[c] = true

	// loop and read from peer initiated that connection
	readPeer(c)
}

func readPeer(c *websocket.Conn) error {
	for {
		_, buf, err := c.ReadMessage()
		if err != nil {
			delete(conns, c)
			return errors.New("connection is down")
		}

		dec := gob.NewDecoder(bytes.NewBuffer(buf))
		var msg Message
		err = dec.Decode(&msg)

		if err != nil {
			return errors.New("decode of peer's message failed")
		}

		log.Println(msg)
		updateFromPeer(msg.e, msg.vc)
	}
}

func updateFromPeer(e crdt.Elem, eVc crdt.VecClock) error {
	// if we can apply the msg now, do so and check the queue
	if vc.caused(eVc) {

	} else {
	}
}

func serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	http.ListenAndServe(addrs[peer], mux)
}

// [0, 1]
// [0, 2]
// [0, 3]

// [0,1]

// [1,1]

// 1 : [0,2], [0,3] , []

// 1 <- [0,1] -> (queue) [0,2] -> (queue) [0,3]

func broadcast(e crdt.Elem, vc crdt.VecClock) {
	for conn := range conns {
		err := conn.WriteMessage()
		if err != nil {
			delete(conns, conn)
		}
	}
}

func main() {
	flag.Parse()
	configString := flag.Arg(1)

	var config Config
	err := json.Unmarshal([]byte(configString), &config)
	if err != nil {
		log.Panic("cannot unmarshal config from flag")
	}

	initialize(config)

	go serve()
}
