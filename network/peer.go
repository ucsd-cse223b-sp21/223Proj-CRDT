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

	"proj/crdt"

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

type Peer struct {
	peer      int
	addrs     []string
	conns     map[*websocket.Conn]bool
	upgrader  websocket.Upgrader
	rga       *crdt.RGA
	broadcast chan crdt.Elem
	gc        chan<- crdt.VecClock
}

func makePeer(c Config) *Peer {
	broadcast := make(chan crdt.Elem)
	rga := crdt.NewRGAOverNetwork(c.peer, len(c.addrs), broadcast)
	peer := Peer{
		peer:      c.peer,
		addrs:     c.addrs,
		upgrader:  websocket.Upgrader{},
		conns:     make(map[*websocket.Conn]bool),
		broadcast: broadcast,
		rga:       rga,
		gc:        crdt.StartGC(rga),
	}

	for i, a := range peer.addrs {
		u := url.URL{Scheme: "ws", Host: a, Path: "/ws"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("connection on join to peer %d failed : %s", i, err)
			continue
		}
		peer.conns[c] = true
		go readPeer(c)
	}

	return &peer
}

func (p *Peer) makeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := p.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("websocket handler failed on upgrade")
			return
		}

		p.conns[c] = true

		// loop and read from peer initiated that connection
		p.readPeer(c)
	}
}

func (p *Peer) readPeer(c *websocket.Conn) error {
	for {
		_, buf, err := c.ReadMessage()
		if err != nil {
			delete(p.conns, c)
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
		p.gc <- vc

		// ignores message if it has already been received
		if !p.rga.Contains(elem) {
			p.broadcast <- elem
			p.rga.Update(elem)
		}
	}
}

func (p *Peer) serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.makeHandler())
	http.ListenAndServe(p.addrs[p.peer], mux)
}

func (p *Peer) writeProc() {
	for {
		e := <-p.broadcast
		msg := Message{e: e, vc: p.rga.VectorClock()}
		for conn := range p.conns {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			enc.Encode(msg)
			e := conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
			if e != nil {
				delete(p.conns, conn)
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

	p := makePeer(config)

	p.serve()
}
