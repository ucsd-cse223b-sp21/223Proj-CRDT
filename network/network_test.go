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
	fmt.Printf("Starting Server at port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        er(err)
    }
	var c Peer.Config
	c.Peer = 0
	c.addr.append("http://localhost:8080")
	Peer.initialize(c)

}
*/

// -addr=localhost:3000

func TestReadPeer(t *testing.T) {
	addrs := []string{"localhost:3000", "localhost:3001"}
	config := Config{
		Peer:  0,
		Addrs: addrs,
	}
	Peer1 := MakePeer(config)
	config.Peer = 1
	Peer2 := MakePeer(config)

	go Peer1.Serve()
	go Peer2.Serve()
	Peer1.InitPeer()
	Peer2.InitPeer()

	elem, err := Peer1.Rga.Append('9', crdt.Id{})
	ne(err)

	time.Sleep(1 * time.Second)

	log.Println(Peer2.Rga.GetView())
	as(Peer2.Rga.Contains(elem))

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
