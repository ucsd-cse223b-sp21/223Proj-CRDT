package network

import (
	"log"
	"proj/crdt"
	"runtime/debug"
	"testing"
	"time"
)

func ne(e error) {
	if e != nil {
		debug.PrintStack()
		log.Fatal(e)
	}
}
func er(e error) {
	if e == nil {
		debug.PrintStack()
		log.Fatal("didn't get an error, when it should")
	}
}
func as(cond bool) {
	if !cond {
		debug.PrintStack()
		log.Fatal("assertion failed")
	}
}

/*func TestInitialize (t *testing.T) {
	fmt.Printf("Starting server at port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        er(err)
    }
	var c peer.Config
	c.peer = 0
	c.addr.append("http://localhost:8080")
	peer.initialize(c)

}
*/

// -addr=localhost:3000

func TestReadPeer(t *testing.T) {
	addrs := []string{"localhost:3000", "localhost:3001"}
	config := Config{
		peer:  0,
		addrs: addrs,
	}
	peer1 := makePeer(config)
	config.peer = 1
	peer2 := makePeer(config)

	go peer1.serve()
	go peer2.serve()
	peer1.initPeer()
	peer2.initPeer()

	elem, err := peer1.rga.Append('9', crdt.Id{})
	ne(err)

	time.Sleep(1 * time.Second)

	log.Println(peer2.rga.GetView())
	as(peer2.rga.Contains(elem))

	//var buf bytes.Buffer
	/*
		//var msg Message
		//e := crdt.Elem{ID: crdt.Id{0, 0, 0}, After: crdt.Id{}, Rem: crdt.Id{}, Val: 8}
		//msg := Message{e: e, vc: crdt.VecClock{}}
		//enc := gob.NewEncoder(&buf)
		//enc.Encode(msg)
		//err := c.WriteMessage(websocket.TextMessage, buf.Bytes())
		if err != nil {
			er(err)
		}
		err1 := readPeer(c)
		ne(err1)
	*/

}
