package network

import (
	"bytes"
	"crdt"
	"encoding/gob"
	"log"
	"runtime/debug"
	"testing"

	"github.com/gorilla/websocket"
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

func TestReadPeer(t *testing.T) {
	var c *websocket.Conn
	var buf bytes.Buffer
	//var msg Message
	e := crdt.Elem{ID: crdt.Id{0, 0, 0}, After: crdt.Id{}, Rem: crdt.Id{}, Val: 8}
	msg := Message{e: e, vc: crdt.VectorClock()}
	enc := gob.NewEncoder(&buf)
	enc.Encode(msg)
	err := c.WriteMessage(websocket.TextMessage, buf.Bytes())
	if err != nil {
		er(err)
	}
	err1 := readPeer(c)
	ne(err1)

}
