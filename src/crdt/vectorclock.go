package crdt

type VecClock struct {
	peer int
	vc   []uint64
}

func newVecClock(peer int, num int) VecClock {
	vc := make([]uint64, num)
	return VecClock{peer, vc}
}

func (v *VecClock) min(b VecClock) VecClock {
	c := *v
	for i := range b.vc {
		if b.vc[i] < c.vc[i] {
			c.vc[i] = b.vc[i]
		}
	}
	c.peer = -1
	return c
}

func (v *VecClock) incrementTo(peer int, seq uint64) {
	if v.vc[peer] < seq {
		v.vc[peer] = seq
	}
}

// true when v is atmost one op behind on b.peer and up-to-date/ahead on everything else
func (v *VecClock) caused(b VecClock) bool {
	caused := true
	for i := range b.vc {
		if b.peer == i && v.vc[i] < (b.vc[i]-1) {
			caused = false
		} else if v.vc[i] < b.vc[i] {
			caused = false
		}
	}

	return caused
}
