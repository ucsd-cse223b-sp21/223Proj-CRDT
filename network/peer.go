package network

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"

	"proj/crdt"

	"github.com/gorilla/websocket"
)

type Config struct {
	Peer  int
	Addrs []string
}

type Message struct {
	E  crdt.Elem
	Vc crdt.VecClock
}

const BACKUP_SIZE = 1000

type Peer struct {
	peer      int
	addrs     []string
	conns     map[*websocket.Conn]bool
	upgrader  websocket.Upgrader
	Rga       *crdt.RGA
	broadcast chan crdt.Elem
	backup    chan crdt.Elem
	gc        chan<- crdt.VecClock
	dc        bool
}

func MakePeer(c Config) *Peer {
	broadcast := make(chan crdt.Elem)
	rga := crdt.NewRGAOverNetwork(c.Peer, len(c.Addrs), broadcast)
	peer := Peer{
		peer:      c.Peer,
		addrs:     c.Addrs,
		upgrader:  websocket.Upgrader{},
		conns:     make(map[*websocket.Conn]bool),
		broadcast: broadcast,
		backup:    make(chan crdt.Elem, BACKUP_SIZE),
		Rga:       rga,
		gc:        crdt.StartGC(rga),
		dc:        false,
	}

	return &peer
}

func (peer *Peer) InitPeer() {
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

	go peer.writeProc()
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
		mT, buf, err := c.ReadMessage()
		log.Printf("Message type: %d", mT)
		log.Printf("Read message on Peer %d", p.peer)

		// TODO: make sure error means disconnection
		if err != nil {
			delete(p.conns, c)
			return errors.New("connection is down")
		}

		log.Println("Read messsage successfully ")

		dec := gob.NewDecoder(bytes.NewBuffer(buf))
		var msg Message
		err = dec.Decode(&msg)

		if err != nil {
			log.Println("Decode failed")
			log.Fatal(err)
			return errors.New("decode of peer's message failed")
		}

		log.Println(msg)
		elem := msg.E
		vc := msg.Vc
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
		if p.dc {
			p.backup <- e
			continue
		} else if len(p.backup) > 0 {
			for len(p.backup) > 0 {
				p.Broadcast(<-p.backup)
			}
		}

		p.Broadcast(e)
	}
}

func (p *Peer) Broadcast(e crdt.Elem) {
	msg := Message{E: e, Vc: p.Rga.VectorClock()}
	buf := bytes.NewBuffer([]byte{})
	enc := gob.NewEncoder(buf)
	enc.Encode(msg)
	by := buf.Bytes()
	for conn := range p.conns {
		e := conn.WriteMessage(websocket.TextMessage, by)
		// TODO make sure error always implies delete
		if e != nil {
			delete(p.conns, conn)
		}
	}
}
