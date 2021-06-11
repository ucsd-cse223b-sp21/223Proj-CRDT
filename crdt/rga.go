package crdt

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
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

type EncodedRGA struct {
	List []Elem
}

type RGA struct {
	Peer     int
	Doc      Document
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

func (r *RGA) GetEncoding() []byte {
	enc := EncodedRGA{
		List: make([]Elem, 0),
	}

	curr := &r.Head
	for curr != nil {
		enc.List = append(enc.List, curr.Elem)
		curr = curr.next
	}

	// log.Printf("Peer %d Encoding Has: %v", r.Peer, enc.List)
	// log.Print ln(r.GetView())

	buf := bytes.NewBuffer([]byte{})
	err := gob.NewEncoder(buf).Encode(enc)
	if err != nil {
		log.Fatal("Encode failure in GetEncoding")
	}
	return buf.Bytes()
}

func (r *RGA) MergeFromEncoding(encBytes []byte) error {
	var enc EncodedRGA
	err := gob.NewDecoder(bytes.NewBuffer(encBytes)).Decode(&enc)
	if err != nil {
		return err
	}

	log.Printf("Decoding Has: %v", enc.List)
	log.Printf("Peer %d Already Has: |%v|", r.Peer, r.GetString())

	for _, enc := range enc.List {
		err = r.Update(enc)
		if err != nil {
			return err
		}
	}

	log.Printf("Peer %d NOW Has: |%v|", r.Peer, r.GetString())
	r.Doc.UpdateView()
	return nil
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
	// return string(b[1:]), i
	return string(b), i
}

func (r *RGA) GetString() string {
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
	r.mut.Lock()
	n, ok := r.m[e.ID]
	r.mut.Unlock()
	if ok && n.Elem.Rem == e.Rem {
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
		vecC: NewVecClock(peer, numPeers),
	}
	r.Doc = NewRgaDoc(&r)

	r.Head.Elem = Elem{ID: Id{0, 0, 0}, After: Id{}, Rem: Id{}, Val: 0}
	r.m[r.Head.Elem.ID] = &r.Head

	return &r
}

func NewRGAOverNetwork(peer int, numPeers int, broadcast chan<- Elem) *RGA {
	r := NewRGA(peer, numPeers)
	r.broadcast = broadcast
	// log.Printf("Broadcast on addition to RGA is %v", broadcast)
	// log.Printf("r.broadcast on addition to RGA is %v", r.broadcast)
	return r
}

func (r *RGA) Length() int {
	L := len(r.m)
	return L
}

// LOCAL OPERATIONS

// appends a new char after an elem by creating a new elem locally
func (r *RGA) Append(val byte, after Id) (Elem, error) {
	e := Elem{ID: r.getNewChange(), After: after, Rem: Id{}, Val: val}

	log.Printf("Appending to rga with broadcast %v, %v", r, r.broadcast)
	// broadcast local change
	if r.broadcast != nil {
		log.Printf("Writing to broadcast at address %p", r.broadcast)
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
		e := Elem{}
		e.Rem = r.getNewChange()
		e.ID = n.Elem.ID
		e.After = n.Elem.After

		// broadcast local change
		if r.broadcast != nil {
			r.broadcast <- e
		}
		return e, r.Update(e)
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
		for j, n := range r.remQ[i] {
			if m >= n.Elem.Rem.Seq {
				if n.next != nil {
					n.next.prev = n.prev
					n.prev.next = n.next
				} else {
					n.prev.next = nil
				}
				r.mut.Lock()
				delete(r.m, n.Elem.ID)
				r.mut.Lock()
			} else {
				r.remQ[i] = r.remQ[i][j:]
				break
			}
		}
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
	// log.Printf("Update on peer %d with elem num %d from peer %d with byte %v", r.Peer, e.ID.Time, e.ID.Peer_, e.Val)
	log.Println("Update beginning with %s with current view '%s'", e.Val, r.Doc.View())

	// node already exists and its being removed (modify node ala tombstone)
	r.mut.Lock()
	n, ok := r.m[e.ID]
	r.mut.Unlock()
	log.Println("Update checked if node exists in map")
	if ok {
		// redundant operation
		if e.Rem.Time == 0 {
			return nil
		}

		n.Elem = e
		// r.remQ = append(r.remQ, n)
		r.remQ[e.Rem.Peer_] = sortedInsert(r.remQ[e.Rem.Peer_], n)
		// update clock/vc for new remove
		r.clock(e.Rem.Time)
		r.vecC.incrementTo(e.Rem.Peer_, e.Rem.Seq)

		if e.Rem.Peer_ != r.Peer {
			r.Doc.UpdateView()
		}
		return nil
	}

	// if parent does not exist, return error (maintains causal order)
	after, ok := r.m[e.After]
	if !ok {
		return errors.New("cannot find parent elem")
	}

	// update clock/vc for new append
	r.clock(e.ID.Time)
	log.Println("e.ID.Peer_", e.ID.Peer_)
	r.vecC.incrementTo(e.ID.Peer_, e.ID.Seq)

	log.Println("Update ready to insert")

	// find insert location
	// DO atomically to avoid data races
	r.mut.Lock()
	// ignore existing node to avoid inconsistency on redundent operations
	_, ok = r.m[e.ID]
	if ok {
		r.mut.Unlock()
		return nil
	}

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
	r.mut.Unlock()

	log.Println("Update insert finished")
	r.Doc.UpdateView()
	// if e.ID.Peer_ != r.Peer {
	// 	r.Doc.UpdateView()
	// }
	log.Println("Update finished")
	return nil
}
