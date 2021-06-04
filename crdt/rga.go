package crdt

import (
	"errors"
	"sync"
)

type Elem struct {
	ID    Id
	After Id
	Rem   Id
	Val   byte
}

type Node struct {
	Elem Elem
	prev *Node
	next *Node
}

// default is "empty" -- valid id's have time > 0
type Id struct {
	Time  uint64
	Peer_ int
	Seq   uint64
}

type RGA struct {
	Peer     int
	numPeers int
	time     uint64
	seq      uint64
	mut      sync.Mutex
	Head     Node
	m        map[Id]*Node
	remQ     [][]*Node
	// remQ      []*Node   // TODO : make gc more efficient with array of arrows indexed by seq
	vecC      VecClock
	broadcast chan<- Elem
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
	return Id{Time: r.time, Peer_: r.Peer}
}

func (r *RGA) GetView() (string, []Id) {
	var b []byte
	var i []Id
	curr := &r.Head
	for curr != nil {
		//if element is not deleted, append character
		if (curr.Elem.Rem == Id{}) {
			b = append(b, curr.Elem.Val)
			i = append(i, curr.Elem.ID)
		}
		curr = curr.next
	}
	return string(b[1:]), i[1:]
}

func (r *RGA) getString() string {
	var b []byte
	curr := &r.Head
	for curr != nil {
		//if element is not deleted, append character
		if (curr.Elem.Rem == Id{}) {
			b = append(b, curr.Elem.Val)
		}
		curr = curr.next
	}
	return string(b[1:])
}

func newRGAList(numPeers int) []*RGA {
	rList := make([]*RGA, numPeers)

	for i := 0; i < 2; i++ {
		rList[i] = NewRGA(i, numPeers)
	}
	return rList
}

func (r *RGA) Contains(e Elem) bool {
	if n, ok := r.m[e.ID]; ok && n.Elem.Rem == e.Rem {
		return true
	}
	return false
}

// create new rga with head node
func NewRGA(peer int, numPeers int) *RGA {
	r := RGA{
		Peer:     peer,
		numPeers: numPeers,
		m:        make(map[Id]*Node),
		// remQ:     make([]*Node, 0),
		remQ: make([][]*Node, numPeers),
		vecC: newVecClock(peer, numPeers),
	}

	r.Head.Elem = Elem{ID: Id{0, 0, 0}, After: Id{}, Rem: Id{}, Val: 0}
	r.m[r.Head.Elem.ID] = &r.Head

	return &r
}

func NewRGAOverNetwork(peer int, numPeers int, broadcast chan<- Elem) *RGA {
	r := NewRGA(peer, numPeers)
	r.broadcast = broadcast
	return r
}

// LOCAL OPERATIONS

// appends a new char after an elem by creating a new elem locally
func (r *RGA) Append(val byte, after Id) (Elem, error) {
	e := Elem{ID: r.getNewChange(), After: after, Rem: Id{}, Val: val}

	// broadcast local change
	if r.broadcast != nil {
		r.broadcast <- e
	}
	return e, r.Update(e)
}

// "removes" an elem by setting its rem field to describe the new operation
func (r *RGA) Remove(id Id) (Elem, error) {
	if id == r.Head.Elem.ID {
		return Elem{}, errors.New("r.head are not removable")
	}
	if n, ok := r.m[id]; ok {
		n.Elem.Rem = r.getNewChange()

		// broadcast local change
		if r.broadcast != nil {
			r.broadcast <- n.Elem
		}
		return n.Elem, nil
	} else {
		return Elem{}, errors.New("cannot remove non-existent node. check local call to remove")
	}
}

// actually deletes "removed" nodes up to id.seq on id.peer (should only be called when all peers are known to have seen it)
func (r *RGA) cleanup(min []uint64) {
	// for i := len(r.remQ) - 1; i >= 0; i-- {
	// 	n := r.remQ[i]

	// 	if min[n.elem.id.peer] >= n.elem.id.seq {
	// 		n.next.prev = n.prev
	// 		n.prev.next = n.next
	// 		delete(r.m, n.elem.id)
	// 		last := len(r.remQ) - 1
	// 		r.remQ[i] = r.remQ[last]
	// 		r.remQ = r.remQ[:last]
	// 	}
	// }
	for i, m := range min {
		r.remQ[i] = r.remQ[i][m:]
	}
}

// determines order of concurrent operations (all other operations are implicitly ordered by clock)
func (e Elem) isNewerThan(e2 Elem) bool {
	a := e.ID
	b := e2.ID
	if a.Time > b.Time {
		return true
	} else if a.Time < b.Time {
		return false
	} else {
		return a.Peer_ < b.Peer_ // no two changes of equal time will have the same peer
	}
}

func (r *RGA) VectorClock() VecClock {
	return r.vecC
}

// binary search and insert
func sortedInsert(list []*Node, node *Node) []*Node {
	l := len(list)
	if l == 0 {
		return append(list, node)
	}

	low := 0
	high := len(list) - 1
	for low < high {
		median := (low + high) / 2
		if list[median].Elem.Rem.Seq < node.Elem.Rem.Seq {
			low = median - 1
		} else if list[median].Elem.Rem.Seq > node.Elem.Rem.Seq {
			high = median + 1
		} else {
			return list
		}
	}

	return append(append(list[:low+1], node), list[low+1:]...)
}

// merge in any elem into RGA (used by local append and any downstream ops)
func (r *RGA) Update(e Elem) error {

	// if node already exists, updates it (maintains idempotency)
	if n, ok := r.m[e.ID]; ok {
		// the remove update is new
		if n.Elem.Rem.Time == 0 && e.Rem.Time != 0 {
			n.Elem = e
			// r.remQ = append(r.remQ, n)
			r.remQ[e.Rem.Peer_] = sortedInsert(r.remQ[e.Rem.Peer_], n)
			// update clock/vc for new remove
			r.clock(e.Rem.Time)
			r.vecC.incrementTo(e.Rem.Peer_, e.Rem.Seq)
		}
		return nil
	}

	// update clock/vc for new append
	r.clock(e.ID.Time)
	r.vecC.incrementTo(e.ID.Peer_, e.ID.Seq)
	// if parent does not exist, return error (maintains causal order)
	after, ok := r.m[e.After]
	if !ok {
		return errors.New("cannot find parent elem")
	}

	// find insert location
	prev := after
	next := prev.next
	for next != nil && next.Elem.After == next.prev.Elem.ID && next.Elem.isNewerThan(e) {
		prev = next
		next = next.next
	}

	node := &Node{Elem: e, next: next, prev: prev}
	if next != nil {
		next.prev = node
	}
	prev.next = node

	r.m[e.ID] = node
	return nil
}
