package crdt

import (
	"log"
	"runtime/debug"
	"testing"
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

func AppendAndUpate(char byte, after Id, r *RGA, rList []*RGA) (Elem, error) {
	elem, err := r.append(char, after)
	if err != nil {
		return Elem{}, err
	}
	err = UpdateAllOtherPeer(r.peer, rList, elem)
	if err != nil {
		return Elem{}, err
	}
	return elem, nil
}

func RemoveAndUpdate(id Id, r *RGA, rList []*RGA) error {
	elem, err := r.remove(id)
	if err != nil {
		return err
	}
	err = UpdateAllOtherPeer(r.peer, rList, elem)
	if err != nil {
		return err
	}
	return nil
}

func AppendStringAndUpdate(text string, after Id, r *RGA, rList []*RGA) ([]Elem, error) {
	elemList := make([]Elem, len(text))
	elemID := after
	for i, char := range text {
		elem, err := AppendAndUpate(byte(char), elemID, r, rList)
		if err != nil {
			return elemList, err
		}
		elemList[i] = elem
		elemID = elem.id

	}
	return elemList, nil
}

func AllPeerViewTest(t *testing.T, rList []*RGA, expect string) {
	for _, r := range rList {
		as(r.getString() == expect)
	}
}

func TestTwoUser(t *testing.T) {
	//init
	numPeers := 2
	viewUpperLimit := 100
	r := newRGAList(numPeers)

	userView := make([][]Elem, numPeers)
	for i := range userView {
		userView[i] = make([]Elem, viewUpperLimit)
	}

	///////////////// Append View test

	//peer 0 type A and expect peer 1 to see A
	elem, err := AppendAndUpate(byte('A'), r[0].head.elem.id, r[0], r)
	ne(err)
	AllPeerViewTest(t, r, "A")

	//peer 1 type B and expect peer 0 to see AB
	//(because on how we sort message prority when before if the same)
	_, err = AppendAndUpate(byte('B'), r[1].head.elem.id, r[1], r)
	ne(err)
	//log.Println(r[0].getString())
	AllPeerViewTest(t, r, "AB")

	//peer 0 types HelloWorld after A, should see
	_, err = AppendStringAndUpdate("HelloWorld", elem.id, r[0], r)
	ne(err)
	//log.Println(r[0].getString())
	AllPeerViewTest(t, r, "AHelloWorldB")

	//////////////// Delete View test

	//peer 1 trys to remove the 'A' peer 0 typed
	err = RemoveAndUpdate(elem.id, r[1], r)
	ne(err)
	AllPeerViewTest(t, r, "HelloWorldB")

}
