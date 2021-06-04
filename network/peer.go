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
	Rga       *crdt.RGA
	broadcast chan crdt.Elem
	gc        chan<- crdt.VecClock
}

func MakePeer(c Config) *Peer {
	broadcast := make(chan crdt.Elem)
	rga := crdt.NewRGAOverNetwork(c.peer, len(c.addrs), broadcast)
	peer := Peer{
		peer:      c.peer,
		addrs:     c.addrs,
		upgrader:  websocket.Upgrader{},
		conns:     make(map[*websocket.Conn]bool),
		broadcast: broadcast,
		Rga:       rga,
		gc:        crdt.StartGC(rga),
	}

	// proactively attempt starting connections on creation
	// if peer goes down and back up, it will attempt to reconnect here
	// (need seperate logic for network partition if we care -- ie: disconnected but not restarted)
	for i, a := range peer.addrs {
		if i == peer.peer {
			continue
		}

		// TODO make sure this url scheme works for connections (based on client example in websocket repo)
		u := url.URL{Scheme: "ws", Host: a, Path: "/ws"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("connection on join to peer %d failed : %s", i, err)
			continue
		}

		// create connection and goroutine for reading from it
		peer.conns[c] = true
		go peer.readPeer(c)
	}

	return &peer
}

// create handler wrapping peer object to read messages from other peer
func (p *Peer) makeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := p.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("websocket handler failed on upgrade")
			return
		}

		// create connection and goroutine for reading from it
		// TODO: determine if need to create go-routine here or not (think we should)
		p.conns[c] = true
		go p.readPeer(c)
	}
}

// reads messages from peer in loop until connection fails
func (p *Peer) readPeer(c *websocket.Conn) error {
	for {
		_, buf, err := c.ReadMessage()
		// TODO: make sure error means disconnection
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
		if !p.Rga.Contains(elem) {
			p.broadcast <- elem
			p.Rga.Update(elem)
		}
	}
}

// have peer start acting as server (can receive websocket connections)
func (p *Peer) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.makeHandler())
	http.ListenAndServe(p.addrs[p.peer], mux)
}

// process for writing(broadcasting) elem's to all peers
// potentially consider parallelizing the writes to different peers?
func (p *Peer) writeProc() {
	for {
		e := <-p.broadcast
		msg := Message{e: e, vc: p.Rga.VectorClock()}
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

// basic main for starting up a peer using config parsed from an argument
func main() {
	flag.Parse()
	configString := flag.Arg(1)

	var config Config
	err := json.Unmarshal([]byte(configString), &config)
	if err != nil {
		log.Panic("cannot unmarshal config from flag")
	}

	p := MakePeer(config)

	p.Serve()
}
