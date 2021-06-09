package network

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

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
	conns     map[int]*websocket.Conn
	upgrader  websocket.Upgrader
	Rga       *crdt.RGA
	broadcast chan crdt.Elem
	backup    chan crdt.Elem
	gc        chan<- crdt.VecClock
	dc        bool
	connect   chan websocket.Conn
}

func MakePeer(c Config) *Peer {
	broadcast := make(chan crdt.Elem)
	rga := crdt.NewRGAOverNetwork(c.Peer, len(c.Addrs), broadcast)
	peer := Peer{
		peer:     c.Peer,
		addrs:    c.Addrs,
		upgrader: websocket.Upgrader{},
		conns:    make(map[int]*websocket.Conn),
		// copyingTo: make(map[int]bool),
		// holding: make(chan []byte, 5),
		broadcast: broadcast,
		backup:    make(chan crdt.Elem, BACKUP_SIZE),
		Rga:       rga,
		gc:        crdt.StartGC(rga),
		dc:        true,
		connect:   make(chan websocket.Conn),
	}

	return &peer
}

func (peer *Peer) initializeFromPeer(conn *websocket.Conn) error {
	first := []byte(strconv.Itoa(peer.peer))
	sendRGA := 0
	if peer.dc {
		sendRGA = 1
	}
	first = append(first, byte(sendRGA))
	err := conn.WriteMessage(websocket.BinaryMessage, first)
	if err != nil {
		return err
	}

	if peer.dc {
		_, buf, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		peer.Rga.MergeFromEncoding(buf)
	}

	return nil
}

func (peer *Peer) initializeOtherPeer(conn *websocket.Conn) (int, error) {
	_, buf, err := conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	sendRga := int(buf[len(buf)-1])

	// send RGA string
	if sendRga == 1 {
		conn.WriteMessage(websocket.BinaryMessage, []byte(peer.Rga.GetEncoding()))
	}

	peerString := string(buf[:len(buf)-1])
	i, err := strconv.Atoi(peerString)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (peer *Peer) connectToPeers() {
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
		log.Printf("connection on join to peer %d succeeded!", i)

		// create connection and goroutine for reading from it
		if peer.conns[i] == nil {
			err = peer.initializeFromPeer(c)
			if err != nil {
				log.Printf("initializeFromPeer failed")
				continue
			}
			log.Printf("Conn set for Old Peer %d on Peer %d", i, peer.peer)
			peer.dc = false
			peer.conns[i] = c
			go peer.readPeer(c, i)
		} else {
			c.Close()
		}
	}

	peer.dc = false
}

func (peer *Peer) InitPeer(handler func(w http.ResponseWriter, r *http.Request)) {
	go peer.serve(handler)
	peer.connectToPeers()
	go peer.writeProc()
}

// create handler wrapping peer object to read messages from other peer
func (p *Peer) makePeerHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if p.dc {
			r.Body.Close()
			return
		}

		c, err := p.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("websocket handler failed on upgrade")
			return
		}

		// create connection and goroutine for reading from it
		// TODO: determine if need to create go-routine here or not (think we should)
		i, err := p.initializeOtherPeer(c)
		if err != nil {
			log.Printf("initializeOtherPeer failed")
			return
		}

		if p.conns[i] == nil {
			p.conns[i] = c
			log.Printf("Conn set for New Peer %d on Peer %d", i, p.peer)
			go p.readPeer(c, i)
		} else {
			c.Close()
		}
	}
}

// reads messages from peer in loop until connection fails
func (p *Peer) readPeer(c *websocket.Conn, ind int) error {
	log.Printf("Reading on Peer %d from Peer %d", p.peer, ind)
	p.Rga.B()
	for {
		_, buf, err := c.ReadMessage()
		// log.Printf("Message type: %d", mT)
		// log.Printf("Read message on Peer %d", p.peer)

		// TODO: make sure error means disconnection
		if err != nil {
			delete(p.conns, ind)
			return errors.New("connection is down")
		}

		// log.Println("Read messsage successfully ")

		dec := gob.NewDecoder(bytes.NewBuffer(buf))
		var msg Message
		err = dec.Decode(&msg)

		if err != nil {
			log.Println("Decode failed")
			log.Fatal(err)
			return errors.New("decode of peer's message failed")
		}

		// log.Println(msg)
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
func (p *Peer) serve(handler func(w http.ResponseWriter, r *http.Request)) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.makePeerHandler())
	if handler != nil {
		mux.HandleFunc("/gui", handler)
	}
	http.ListenAndServe(p.addrs[p.peer], mux)
}

func (p *Peer) Disconnect() {
	for c := range p.conns {
		p.conns[c].Close()
	}
	p.conns = make(map[int]*websocket.Conn)
	p.dc = true
}

// start peer from
func (p *Peer) Connect() {
	p.connectToPeers()
	p.broadcast <- crdt.Elem{}
}

// // start peer from
// func (p *Peer) managePeer() {
// 	for {
// 		c := <-p.connect
// 		p.initializeFromPeer(c)
// 	}
// }

// process for writing(broadcasting) elem's to all peers
// potentially consider parallelizing the writes to different peers?
func (p *Peer) writeProc() {
	for {
		log.Printf("WriteProc running with broadcast at address %p", p.broadcast)
		e := <-p.broadcast
		log.Printf("Writing element from Peer %d", p.peer)
		log.Printf("p.dc: %t", p.dc)
		log.Printf("len(p.backup): %d", len(p.backup))
		// if p.dc || len(p.conns) == 0 {
		// p.dc = true
		if p.dc {
			p.backup <- e
			continue
		} else if len(p.backup) > 0 {
			for len(p.backup) > 0 {
				b := <-p.backup
				p.Rga.Update(b)
				p.Broadcast(b)
			}
		}

		if e.ID.Time != 0 {
			p.Broadcast(e)
		}
	}
}

func (p *Peer) Broadcast(e crdt.Elem) {
	log.Printf("Broadcasting element from Peer %d", p.peer)
	msg := Message{E: e, Vc: p.Rga.VectorClock()}
	buf := bytes.NewBuffer([]byte{})
	enc := gob.NewEncoder(buf)
	enc.Encode(msg)
	by := buf.Bytes()

	// // for messages send after Rga copy sent to new peer i but before the p.conns[i] is set
	// for k := range p.copyingTo {
	// 	p.holding[k] <- by
	// }
	for k := range p.conns {
		// // p.conns[i] has been set
		// if p.copyingTo[k] {
		// 	delete(p.copyingTo, k)
		// 	for len(p.holding) > 0 {
		// 		b := <-p.holding[k]
		// 		e := p.conns[k].WriteMessage(websocket.TextMessage, b)
		// 		if e != nil {
		// 			delete(p.conns, k)
		// 			// clear them
		// 			p.copyingTo =
		// 			p.holding =
		// 		}
		// 	}
		// }

		e := p.conns[k].WriteMessage(websocket.TextMessage, by)
		// TODO make sure error always implies delete
		if e != nil {
			delete(p.conns, k)
		}
	}
}
