package network

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"net/url"

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
var rga *crdt.RGA
var broadcast chan Message
var gc chan<- crdt.VecClock

func initialize(c Config) {
	peer = c.peer
	addrs = c.addrs

	conns = make(map[*websocket.Conn]bool)
	broadcast = make(chan Message)
	rga = crdt.NewRGAOverNetwork(peer, len(addrs), broadcast)
	gc = crdt.StartGC(rga)

	for i, a := range addrs {
		u := url.URL{Scheme: "ws", Host: a, Path: "/ws"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("connection on join to peer %d failed : %s", i, err)
			continue
		}
		conns[c] = true
		go readPeer(c)
	}
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
		elem := msg.e
		vc := msg.vc
		gc <- vc

		// ignores message if it has already been received
		if !rga.Contains(elem) {
			broadcast <- Message{e: elem, vc: rga.VectorClock()}
			rga.Update(elem)
		}
	}
}

func serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	http.ListenAndServe(addrs[peer], mux)
}

func writeProc() {
	for {
		msg := <-broadcast
		for conn := range conns {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			enc.Encode(msg)
			e := conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
			if e != nil {
				delete(conns, conn)
			}
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

	serve()
}
