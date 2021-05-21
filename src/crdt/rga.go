package crdt

import (
	"errors"
	"log"
	"sync"
)

type Elem struct {
	id    Id
	after Id
	rem   Id
	val   byte
}

type Node struct {
	elem Elem
	prev *Node
	next *Node
}

// default is "empty" -- valid id's have time > 0
type Id struct {
	time uint64
	peer int
	seq  uint64
}

type RGA struct {
	peer     int
	numPeers int
	time     uint64
	seq      uint64
	mut      sync.Mutex
	head     Node
	m        map[Id]*Node
	// remQ [][]*Node
	remQ []*Node
}

func (r *RGA) clock(atLeast uint64) {
	r.mut.Lock()
	if atLeast > r.time {
		r.time = atLeast
	}
	r.time = r.time + 1
	r.mut.Unlock()
}

// makes every local operation on the rga unique by incrementing the clock
func (r *RGA) getNewChange() Id {
	r.clock(0)
	r.seq = r.seq + 1
	return Id{time: r.time, peer: r.peer}
}

func (r *RGA) getString() string {
	var b []byte
	curr := &r.head
	for curr != nil {
		//if element is not deleted, append character
		if (curr.elem.rem == Id{}) {
			b = append(b, curr.elem.val)
		}
		curr = curr.next
	}
	return string(b[1:])
}

func newRGAList(numPeers int) []*RGA {
	rList := make([]*RGA, numPeers)

	for i := 0; i < 2; i++ {
		rList[i] = newRGA(i, numPeers)
	}
	return rList
}

// create new rga with head node
func newRGA(peer int, numPeers int) *RGA {
	r := RGA{}
	r.peer = peer
	r.numPeers = numPeers

	r.head = Node{
		elem: Elem{id: Id{0, 0, 0}, after: Id{}, rem: Id{}, val: 0},
		next: nil,
		prev: nil}
	r.m = make(map[Id]*Node)
	r.m[r.head.elem.id] = &r.head

	return &r
}

// LOCAL OPERATIONS

// appends a new char after an elem by creating a new elem locally
func (r *RGA) append(val byte, after Id) (Elem, error) {
	e := Elem{id: r.getNewChange(), after: after, rem: Id{}, val: val}
	return e, r.update(e)
}

// "removes" an elem by setting its rem field to describe the new operation
func (r *RGA) remove(id Id) (Elem, error) {
	if id == r.head.elem.id {
		return Elem{}, errors.New("r.head are not removable")
	}
	if n, ok := r.m[id]; ok {
		n.elem.rem = r.getNewChange()
		return n.elem, nil
	} else {
		return Elem{}, errors.New("cannot remove non-existent node. check local call to remove")
	}
}

// actually deletes "removed" nodes up to id.seq on id.peer (should only be called when all peers are known to have seen it)
func (r *RGA) cleanup(min []uint64) {
	for i := len(r.remQ) - 1; i >= 0; i-- {
		n := r.remQ[i]

		if min[n.elem.id.peer] >= n.elem.id.seq {
			n.next.prev = n.prev
			n.prev.next = n.next
			delete(r.m, n.elem.id)
			last := len(r.remQ) - 1
			r.remQ[i] = r.remQ[last]
			r.remQ = r.remQ[:last]
		}
	}
}

// determines order of concurrent operations (all other operations are implicitly ordered by clock)
func (e Elem) isNewerThan(e2 Elem) bool {
	a := e.id
	b := e2.id
	if a.time > b.time {
		return true
	} else if a.time < b.time {
		return false
	} else {
		return a.peer < b.peer // no two changes of equal time will have the same peer
	}
}

// merge in any elem into RGA (used by local append and any downstream ops)
func (r *RGA) update(e Elem) error {

	// if node already exists, updates it (maintains idempotency)
	if n, ok := r.m[e.id]; ok {
		if e.rem.time != 0 {
			n.elem = e
			r.remQ = append(r.remQ, n)
		}
	}
	log.Println("updating peer", r.peer, " after", e.after)
	// if parent does not exist, return error (maintains causal order)
	after, ok := r.m[e.after]
	if !ok {
		return errors.New("cannot find parent elem")
	}

	// find insert location
	prev := after
	next := prev.next
	for next != nil && next.elem.after == next.prev.elem.id && next.elem.isNewerThan(e) {
		prev = next
		next = next.next
	}

	node := &Node{elem: e, next: next, prev: prev}
	if next != nil {
		next.prev = node
	}
	prev.next = node

	r.m[e.id] = node
	return nil
}
