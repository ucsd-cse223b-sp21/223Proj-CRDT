package document

import (
	"encoding/json"
	"flag"
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

func getPeer() *network.Peer {
	flag.Parse()
	configString := flag.Arg(1)

	var config network.Config
	err := json.Unmarshal([]byte(configString), &config)
	if err != nil {
		log.Panic("cannot unmarshal config from flag")
	}

	p := network.MakePeer(config)

	p.Serve()
	return p
}

func TestDocTwoUser(t *testing.T) {
	// create doc for 2
	numPeer := 2
	doc := make([]RgaDoc, numPeer)
	for i := 0; i < numPeer; i++ {
		peer := getPeer()
		doc[i] = *NewRgaDoc(peer.Rga)

		as(doc[i].View() == "")
	}

	typeThis(&doc[0], 0, "HELLOWORLD!")
	typeThis(&doc[1], 0, "helloWorld!")

	time.Sleep(500 * time.Millisecond)

	log.Println(doc[0].View())
	log.Println(doc[1].View())
	time.Sleep(ViewProTime)
	as(doc[0].View() == doc[1].View())
}

func typeThis(doc *RgaDoc, cursor int, text string) {
	for _, cha := range text {
		err := doc.Append(cursor, byte(cha))
		cursor++
		ne(err)
	}
}

func TestDocLimit(t *testing.T) {

}
