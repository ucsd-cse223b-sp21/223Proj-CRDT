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
}

func TestFaultTolerance(t *testing.T) {
	addrs := []string{"localhost:8080", "localhost:8081"}
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

	Peer1.dc = true
	elem, err := Peer1.Rga.Append('1', crdt.Id{})
	ne(err)
	elem1, err1 := Peer1.Rga.Append('r', crdt.Id{})
	elem_r, err_r := Peer1.Rga.Remove(elem.ID)
	ne(err_r)
	Peer1.dc = false
	time.Sleep(2 * time.Second)

	log.Println(Peer2.Rga.GetView())
	as(Peer2.Rga.Contains(elem))

}
