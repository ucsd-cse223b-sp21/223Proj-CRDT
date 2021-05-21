package crdt

import (
	"log"
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

func TestSingleUser(t *testing.T) {
	// creating new rga
	r := newRGA(1, 1)
	as(r.getString() == "")

	//attempt to remove head
	_, err := r.remove(r.head.elem.id)
	er(err)
	as(r.getString() == "")

	//typing '123'
	elem, err := r.append(byte('1'), r.head.elem.id)
	ne(err)
	as(r.getString() == "1")
	elem, err = r.append(byte('2'), elem.id)
	ne(err)
	elem, err = r.append(byte('3'), elem.id)
	ne(err)

	//rga should contain '123'
	as(r.getString() == "123")

	//single remove
	_, err = r.remove(elem.id)
	ne(err)

	as(r.getString() == "12")

	//double deleting the same element
	_, err = r.remove(elem.id)
	ne(err)

	as(r.getString() == "12")
}

func OneInManyRGABasicTest(t *testing.T, r *RGA) {

	//attempt to remove head
	_, err := r.remove(r.head.elem.id)
	er(err)
}

func UpdateAllOtherPeer(peer int, rgaList []*RGA, elem Elem) error {
	for i, r := range rgaList {
		if i == peer {
			continue
		}
		err := r.update(elem)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestTwoUser(t *testing.T) {
	numPeers := 2
	viewUpperLimit := 100
	r := newRGAList(numPeers)

	userView := make([][]Elem, numPeers)
	for i := range userView {
		userView[i] = make([]Elem, viewUpperLimit)
	}

	elem, err := r[0].append(byte('A'), r[0].head.elem.id)
	ne(err)
	userView[0][0] = elem
	ne(UpdateAllOtherPeer(0, r, elem))

	time.Sleep(1 * time.Second)
	//rga should contain '123'
	log.Println(r[0].getString())
	log.Println(r[1].getString())
	as(r[1].getString() == "A")
}
