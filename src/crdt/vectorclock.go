package crdt

type VecClock struct {
	peer int
	vc   []uint64
}
