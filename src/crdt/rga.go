package crdt

import (
	"errors"
	"network"
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
	// remQ [][]*Node // TODO : make gc more efficient
	remQ      []*Node
	vecC      VecClock
	broadcast chan<- network.Message
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
		b = append(b, curr.elem.val)
		curr = curr.next
	}
	return string(b)
}

func (r *RGA) Contains(e Elem) bool {
	if n, ok := r.m[e.id]; ok && n.elem.rem == e.rem {
		return true
	}
	return false
}

// create new rga with head node
func NewRGA(peer int, numPeers int) *RGA {
	r := RGA{
		peer:     peer,
		numPeers: numPeers,
		m:        make(map[Id]*Node),
		remQ:     make([]*Node, 0),
		vecC:     newVecClock(peer, numPeers),
	}

	r.head.elem = Elem{id: r.getNewChange(), after: Id{}, rem: Id{}, val: 0}
	r.m[r.head.elem.id] = &r.head

	return &r
}

func NewRGAOverNetwork(peer int, numPeers int, broadcast chan<- network.Message) *RGA {
	r := NewRGA(peer, numPeers)
	r.broadcast = broadcast
	return r
}

// LOCAL OPERATIONS

// appends a new char after an elem by creating a new elem locally
func (r *RGA) append(val byte, after Id) (Elem, error) {
	e := Elem{id: r.getNewChange(), after: after, rem: Id{}, val: val}

	// broadcast local change
	if r.broadcast != nil {
		r.broadcast <- network.Message{e: e, vc: r.VectorClock()}
	}
	return e, r.Update(e)
}

// "removes" an elem by setting its rem field to describe the new operation
func (r *RGA) remove(id Id) (Elem, error) {
	if n, ok := r.m[id]; ok {
		n.elem.rem = r.getNewChange()

		// broadcast local change
		if r.broadcast != nil {
			r.broadcast <- network.Message{e: n.elem, vc: r.VectorClock()}
		}
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

func (r *RGA) VectorClock() VecClock {
	return r.vecC
}

// merge in any elem into RGA (used by local append and any downstream ops)
func (r *RGA) Update(e Elem) error {

	// if node already exists, updates it (maintains idempotency)
	if n, ok := r.m[e.id]; ok {
		// the remove update is new
		if n.elem.rem.time == 0 && e.rem.time != 0 {
			n.elem = e
			r.remQ = append(r.remQ, n)
			// update clock/vc for new remove
			r.clock(e.rem.time)
			r.vecC.incrementTo(e.rem.peer, e.rem.seq)
		}
		return nil
	}

	// update clock/vc for new append
	r.clock(e.id.time)
	r.vecC.incrementTo(e.id.peer, e.id.seq)

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
