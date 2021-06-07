package crdt

type VecClock struct {
	Peer int
	Vc   []uint64
}

func NewVecClock(peer int, num int) VecClock {
	vc := make([]uint64, num)
	return VecClock{peer, vc}
}

func (v *VecClock) min(b VecClock) VecClock {
	c := *v
	for i := range b.Vc {
		if b.Vc[i] < c.Vc[i] {
			c.Vc[i] = b.Vc[i]
		}
	}
	c.Peer = -1
	return c
}

func (v *VecClock) incrementTo(peer int, seq uint64) {
	if v.Vc[peer] < seq {
		v.Vc[peer] = seq
	}
}

// true when v is atmost one op behind on b.peer and up-to-date/ahead on everything else
func (v *VecClock) caused(b VecClock) bool {
	caused := true
	for i := range b.Vc {
		if b.Peer == i && v.Vc[i] < (b.Vc[i]-1) {
			caused = false
		} else if v.Vc[i] < b.Vc[i] {
			caused = false
		}
	}

	return caused
}
