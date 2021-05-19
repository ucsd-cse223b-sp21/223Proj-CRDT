package crdt

import (
	"errors"
	"sync"
)

type Elem struct {
	id    Id
	after Id
	rem   Id
	val   byte
}

type Node struct {
	elem   Elem
	after  *Node
	before *Node
}

// default is "empty" -- valid id's have time > 0
type Id struct {
	time uint64
	peer int
	seq  uint64
}

type RGA struct {
	peer     int
	time     uint64
	seq      uint64
	mut      sync.Mutex
	head     Node
	m        map[Id]*Node
}

// makes every local operation on the rga unique by incrementing the clock
func (r *RGA) getNewChange() Id {
	r.mut.Lock()
	r.time = r.time + 1
	r.mut.Unlock()
	r.seq = r.seq + 1
	return Id{time: r.time, peer: r.peer}
}

// create new rga with head node
func newRGA(peer int) *RGA {
	r := RGA{}
	r.peer = peer

	r.head = Node{
		elem:   Elem{id: r.getNewChange(), after: Id{}, rem: Id{}, val: 0},
		after:  nil,
		before: nil}

	return &r
}

func (r *RGA) append(val byte, after Id) (Elem, error) {
	e := Elem{id: r.getNewChange(), after: after, rem: Id{}, val: val}
	return e, r.update(e)
}

func (r *RGA) remove(id Id) (Elem, error) {
	if n, ok := r.m[id]; ok {
		n.elem.rem = r.getNewChange()
		return n.elem, nil
	} else {
		return Elem{}, errors.New("cannot remove non-existent node. check local call to remove")
	}
}

func (r *RGA) delete(id Id) error {
	if n, ok := r.m[id]; ok {
		if n.elem.rem == (Id{}) {
			return errors.New("cannot delete non-removed node. check local call to delete")
		}

		n.before.after = n.after
		n.after.before = n.before
		delete(r.m, id)
		return nil
	} else {
		return errors.New("cannot delete non-existent node. check gc call to delete")
	}
}

func (r *RGA) clock(atLeast uint64) {
	r.mut.Lock()
	if atLeast > r.time {
		r.time = atLeast
	}
	r.mut.Unlock()
}

func (r *RGA) update(e Elem) error {
	// if node already exists, updates it (maintains idempotency)
	if n, ok := r.m[e.id]; ok {
		if e.rem.time == 0 {
			n.elem = e
		}
	}

	// if parent does not exist, return error (maintains causal order)
	after, ok := r.m[e.after]
	if !ok {
		return errors.New("Cannot find parent char")
	}

	node := &Node{elem: e, after: after, before: after.before}

	if after.before != nil {
		after.before.after = node
	}
	after.before = node

	return nil
}
