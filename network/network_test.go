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

func TestReadPeers(t *testing.T) {
	addrs := []string{"localhost:3000", "localhost:3001", "localhost:3002", "localhost:4000", "localhost:4004", "localhost:4040", "localhost:8000", "localhost:8080", "localhost:8008", "localhost:3050",
		"localhost:3004", "localhost:3005", "localhost:3006", "localhost:4001", "localhost:4005", "localhost:4080", "localhost:8040", "localhost:8090", "localhost:8009", "localhost:3090"}
	config := Config{
		Peer:  0,
		Addrs: addrs,
	}
	//Peer1 := MakePeer(config)
	peer_list := make([]*Peer, len(addrs))
	for i := 0; i < len(addrs); i++ {
		config.Peer = i
		peer_list[i] = MakePeer(config)
		// go peer_list[i].Serve()
		peer_list[i].InitPeer()
	}

	elem, err := peer_list[0].Rga.Append('9', crdt.Id{})
	ne(err)
	/*
		_, err2 := peer_list[0].Rga.Append('8', crdt.Id{})
		ne(err2)
		_, err1 := peer_list[0].Rga.Remove(elem.ID)
		ne(err1)

		s := peer_list[0].Rga.GetString()

	*/
	time.Sleep(2 * time.Second)

	for i := 1; i < len(addrs); i++ {
		log.Println(peer_list[i].Rga.GetView())
		as(peer_list[i].Rga.Contains(elem))
	}

}

func TestFaultTolerance(t *testing.T) {

	addrs := []string{"localhost:5000", "localhost:5001", "localhost:5002", "localhost:6000", "localhost:6004", "localhost:6040", "localhost:6000", "localhost:6080", "localhost:6008", "localhost:6050",
		"localhost:5004", "localhost:5005", "localhost:5006", "localhost:6001", "localhost:6005", "localhost:7080", "localhost:7040", "localhost:7090", "localhost:7009", "localhost:7090"}

	config := Config{
		Peer:  0,
		Addrs: addrs,
	}
	peer_list := make([]*Peer, len(addrs))
	for i := 0; i < len(addrs); i++ {
		config.Peer = i
		peer_list[i] = MakePeer(config)
		// go peer_list[i].Serve()
		peer_list[i].InitPeer()
	}

	peer_list[0].dc = true
	elem, err := peer_list[0].Rga.Append('1', crdt.Id{})
	ne(err)
	//elem1, err1 := Peer1.Rga.Append('r', crdt.Id{})
	//elem_r, err_r := Peer1.Rga.Remove(elem.ID)
	//ne(err_r)
	peer_list[0].dc = false
	time.Sleep(2 * time.Second)

	for i := 1; i < len(addrs); i++ {
		log.Println(peer_list[i].Rga.GetView())
		as(peer_list[i].Rga.Contains(elem))
	}
}

func TestShortestLocalTime(t *testing.T) {
	addrs := []string{"localhost:3000", "localhost:3001", "localhost:3002", "localhost:3003", "localhost:3004",
		"localhost:3005", "localhost:3006", "localhost:3007", "localhost:3008", "localhost:3009", "localhost:3010"}

	config := Config{
		Peer:  0,
		Addrs: addrs,
	}

	log.Println("in")

	peer_list := make([]*Peer, len(addrs))
	for i := 0; i < len(addrs); i++ {
		config.Peer = i
		peer_list[i] = MakePeer(config)
		//go peer_list[i].Serve()
		peer_list[i].InitPeer()
	}

	for j := 500; j > 0; j = j - 10 {
		log.Println("time = ", j, "ms")
		elem, err := peer_list[0].Rga.Append('1', crdt.Id{})
		ne(err)
		time.Sleep(time.Duration(j) * time.Millisecond)

		for i := 1; i < len(addrs); i++ {
			log.Println(peer_list[i].Rga.GetView())
			as(peer_list[i].Rga.Contains(elem))
		}
	}
}
