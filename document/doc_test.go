package document

import (
	"fmt"
	"log"
	"math"
	"proj/crdt"
	"proj/network"
	"runtime/debug"
	"testing"
	"time"
)

// Here are some promise we made in proposal
const ViewProTime = 500 * time.Millisecond     // time when change from other should be updated
const LocalViewProTime = 50 * time.Millisecond //time when local view should be updated
const CharRateLimit = int(400 / 60)            //400 character per min
const MaxUser = 10                             //max concurrent users
var MaxCharFile = 10 * math.Pow(1024, 2)       //10MB

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

// func TestDoc(t *testing.T) {
// 	log.Println("testing document")

// 	// create doc
// 	doc_pointer := new(NiaveDoc)
// 	var doc Document = *doc_pointer
// 	as(doc.View() == "")

// 	//append out of range test
// 	_, err := doc.Append(-1, byte('n'))
// 	er(err)
// 	_, err = doc.Append(len(doc.View())+1, byte('n'))
// 	er(err)

// 	// append view test
// 	cursor := 0
// 	for _, cha := range "Hello World!" {
// 		newdoc, err := doc.Append(cursor, byte(cha))
// 		doc = newdoc
// 		cursor++
// 		ne(err)
// 	}
// 	log.Println(doc.View())
// 	as(doc.View() == "Hello World!")

// 	// remove out of range test
// 	_, err = doc.Remove(-1)
// 	er(err)
// 	_, err = doc.Remove(len("Hello World!") + 1)
// 	er(err)

// 	// remove view test
// 	doc, err = doc.Remove(len("Hello World!") - 1)
// 	ne(err)
// 	as(doc.View() == "Hello World")

// 	doc, err = doc.Remove(0)
// 	ne(err)
// 	doc, err = doc.Remove(4)
// 	ne(err)
// 	as(doc.View() == "elloWorld")

// }

func TestRgaDoc(t *testing.T) {
	log.Println("testing document")

	// create doc
	r := crdt.NewRGA(0, 1)
	doc := *NewRgaDoc(r)

	as(doc.View() == "")

	//append out of range test
	err := doc.Append(-1, byte('n'))
	er(err)
	err = doc.Append(len(doc.View())+1, byte('n'))
	er(err)

	// append view test
	cursor := 0
	for _, cha := range "Hello World!" {
		err := doc.Append(cursor, byte(cha))
		cursor++
		ne(err)
	}
	log.Println(doc.View())
	as(doc.View() == "Hello World!")

}

func TestPeer2(t *testing.T) {
	// getting 10 peers
	addrs := []string{}
	for i := 0; i < 10; i++ {
		addrs = append(addrs, fmt.Sprintf("localhost:310%d", i))
	}

	config := network.Config{
		Peer:  0,
		Addrs: addrs,
	}

	Peer := make([]*network.Peer, 10)
	doc := make([]*RgaDoc, 10)

	for i := 0; i < 10; i++ {
		config.Peer = i
		Peer[i] = network.MakePeer(config)
		go Peer[i].Serve()
		Peer[i].InitPeer()
		doc[i] = NewRgaDoc(Peer[i].Rga)
	}

	typeThis(doc[0], 0, "HELLOWORLD!")
	typeThis(doc[1], 0, "helloWorld!")

	time.Sleep(ViewProTime)
	doc[0].UpdateView()
	doc[1].UpdateView()

	log.Println(doc[0].View())
	log.Println(doc[1].View())
	time.Sleep(ViewProTime)
	as(doc[0].View() != "")
	as(doc[1].View() != "")
	as(doc[0].View() == doc[1].View())
}

func typeThis(doc *RgaDoc, cursor int, text string) {
	for _, cha := range text {
		err := doc.Append(cursor, byte(cha))
		cursor++
		ne(err)
	}
}

func TestDocDisconnect(t *testing.T) {
	// getting 10 peers
	addrs := []string{}
	for i := 0; i < 10; i++ {
		addrs = append(addrs, fmt.Sprintf("localhost:320%d", i))
	}

	config := network.Config{
		Peer:  0,
		Addrs: addrs,
	}

	Peer := make([]*network.Peer, 10)
	doc := make([]*RgaDoc, 10)

	for i := 0; i < 10; i++ {
		config.Peer = i
		Peer[i] = network.MakePeer(config)
		go Peer[i].Serve()
		Peer[i].InitPeer()
		doc[i] = NewRgaDoc(Peer[i].Rga)
	}

	typeThis(doc[0], 0, "base")
	typeThis(doc[1], 0, "Before_disconnect_")

	//Peer[1].Dc = true
	typeThis(doc[1], 0, "After_disconnect_")
	AllDocUpdateView(doc, true)
	log.Println("doc[1] sees", doc[1].View())
	as(doc[1].View() == "After_disconnect_Before_disconnect_base")
}

func AllDocUpdateView(docList []*RgaDoc, wait bool) {
	if wait {
		time.Sleep(ViewProTime)
	}
	for _, doc := range docList {
		doc.UpdateView()
	}
}

func TestDocLimit(t *testing.T) {

}
