package crdt

func minVC(nums ...VecClock) VecClock {
	c := nums[0]
	for _, v := range nums {
		c = c.min(v)
	}
	return c
}

// creates channel
func startGC(r *RGA) chan<- VecClock {
	c := make(chan VecClock)

	go func() {
		vcs := make([]VecClock, r.numPeers)

		for {
			vc := <-c
			vcs[vc.peer] = vc

			// get min knowledge across peers
			min := minVC(vcs...)
			r.cleanup(min.vc)
		}
	}()

	return c
}
