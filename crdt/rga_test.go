package crdt

import (
	"log"
	"testing"
)

func TestSingleUser(t *testing.T) {
	// creating new rga
	r := NewRGA(0, 1)

	//as(r.Length() == 0)

	as(r.GetString() == "")

	//attempt to remove head
	_, err := r.Remove(r.Head.Elem.ID)
	er(err)
	as(r.GetString() == "")

	//typing '123'
	elem, err := r.Append(byte('1'), r.Head.Elem.ID)
	ne(err)
	as(r.GetString() == "1")
	elem, err = r.Append(byte('2'), elem.ID)
	ne(err)
	elem, err = r.Append(byte('3'), elem.ID)
	ne(err)

	//rga should contain '123'
	as(r.GetString() == "123")

	//single remove
	_, err = r.Remove(elem.ID)

	ne(err)

	as(r.GetString() == "12")

	//double deleting the same element
	_, err = r.Remove(elem.ID)
	ne(err)

	as(r.GetString() == "12")
}

func UpdateAllOtherPeer(peer int, rgaList []*RGA, elem Elem) error {
	for i, r := range rgaList {
		if i == peer {
			continue
		}
		err := r.Update(elem)
		if err != nil {
			return err
		}
	}
	return nil
}

func AppendAndUpate(char byte, after Id, r *RGA, rList []*RGA) (Elem, error) {
	elem, err := r.Append(char, after)
	if err != nil {
		return Elem{}, err
	}
	err = UpdateAllOtherPeer(r.Peer, rList, elem)
	if err != nil {
		return Elem{}, err
	}
	return elem, nil
}

func RemoveAndUpdate(id Id, r *RGA, rList []*RGA) error {
	elem, err := r.Remove(id)
	if err != nil {
		return err
	}
	err = UpdateAllOtherPeer(r.Peer, rList, elem)
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
		elemID = elem.ID
	}
	return elemList, nil
}

func AllPeerViewTest(t *testing.T, rList []*RGA, expect string) {
	for _, r := range rList {
		log.Println("r.GetString()", r.GetString())
		as(r.GetString() == expect)
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
	elem, err := AppendAndUpate(byte('A'), r[1].Head.Elem.ID, r[0], r)
	ne(err)
	AllPeerViewTest(t, r, "A")

	//peer 1 type B and expect peer 0 to see AB
	//(because on how we sort message prority when before if the same)
	_, err = AppendAndUpate(byte('B'), r[0].Head.Elem.ID, r[1], r)
	ne(err)
	//log.Println(r[0].getString())
	AllPeerViewTest(t, r, "BA")

	//peer 0 types HelloWorld after A, should see
	_, err = AppendStringAndUpdate("HelloWorld", elem.ID, r[0], r)
	ne(err)
	//log.Println(r[0].getString())
	AllPeerViewTest(t, r, "BAHelloWorld")

	//////////////// Delete View test

	//peer 1 trys to remove the 'A' peer 0 typed
	err = RemoveAndUpdate(elem.ID, r[1], r)
	ne(err)
	AllPeerViewTest(t, r, "BHelloWorld")

}
