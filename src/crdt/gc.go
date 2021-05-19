package crdt

import (
	"math"
)

func min(nums ...uint64) uint64 {
	var min uint64 = math.MaxUint64
	for _, v := range nums {
		if v < min {
			min = v
		}
	}
	return min
}

// creates channel
func startGC(r *RGA) chan<- VecClock {
	c := make(chan VecClock)

	go func() {
		vcs := make([][]uint64, 0)
		for {
			vc := <-c
			vcs[vc.peer] = vc.vc

			// get min knowledge across peers
			min := make([]uint64, len(vcs[0]))
			for i := range min {
				var m uint64 = math.MaxUint64
				for _, c := range vcs {
					if c[i] < m {
						m = c[i]
					}
				}
				min[i] = m
			}

			r.cleanup(min)
		}
	}()

	return c
}
