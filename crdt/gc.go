package crdt

func minVC(nums ...VecClock) VecClock {
	c := nums[0]
	for _, v := range nums {
		c = c.min(v)
	}
	return c
}

// creates channel
func StartGC(r *RGA) chan<- VecClock {
	c := make(chan VecClock)

	go func() {
		vcs := make([]VecClock, r.numPeers)
		for i := range vcs {
			vcs[i] = NewVecClock(i, r.numPeers)
		}

		for {
			vc := <-c
			vcs[vc.Peer] = vc

			// get min knowledge across peers
			min := minVC(vcs...)
			r.cleanup(min.Vc)
		}
	}()

	return c
}
