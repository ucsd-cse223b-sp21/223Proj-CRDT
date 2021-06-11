package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"proj/crdt"
	"proj/network"
	"strconv"
	"time"
)

var (
	numPeers = flag.Int("numPeers", 2, "number of peers (max 8)")
	peers    = make([]*network.Peer, 0)
	config   = network.Config{
		Addrs: []string{
			"localhost:3001",
			"localhost:3002",
			"localhost:3003",
			"localhost:3004",
			"localhost:3005",
			"localhost:3006",
			"localhost:3007",
			"localhost:3008",
		},
	}
)

// basic main for starting up a peer using config parsed from an argument
func main() {
	network.BENCH = true
	flag.Parse()

	config.Addrs = config.Addrs[:*numPeers]

	nF, err := os.Create("Nlatency.csv")
	lF, err := os.Create("Llatency.csv")
	mF, err := os.Create("memory.csv")
	if err != nil {
		log.Fatal(err)
	}
	nF.Truncate(0)
	lF.Truncate(0)
	mF.Truncate(0)
	nW := csv.NewWriter(nF)
	lW := csv.NewWriter(lF)
	mW := csv.NewWriter(mF)

	// vary proportion of appends to removes to get performance data
	var p float64
	for p = 0.8; p > 0.5; p -= 0.00001 {
		Lat := make(chan float64, 100)
		setupPeers(*numPeers, Lat)
		log.Printf("peer 1 is at address %p and peer 2 at address %p", peers[0], peers[1])

		//var nLat float64
		var lLat float64
		for t := 0; t < 100; t++ {
			log.Println("T is :", t)

			// seed with varying number
			rand.Seed(time.Now().Unix())
			// append on random peer and take average latency across others ????
			pInd := rand.Intn(*numPeers)

			var d crdt.Document
			d = peers[pInd].Rga.Doc

			shouldRem := rand.Float64() > p && d.Length() > 0
			if shouldRem {
				dInd := rand.Intn(d.Length()) + 1 // (0,length]
				// remove
				start := time.Now()
				d.Remove(dInd)
				lLat = time.Since(start).Seconds()
			} else {
				dInd := rand.Intn(d.Length() + 1)
				val := byte('a' + rand.Intn(26))
				// val := byte(rand.Intn(256))
				log.Println("appending ", val)

				start := time.Now()
				d.Append(dInd, val)
				lLat = time.Since(start).Seconds()

				// time.Sleep(100 * time.Millisecond)
				log.Printf("view is '%s'", d.View())
			}

			// todo add the actual logging that generates the values
			total := 0.0
			for i := 0; i < *numPeers-1; i++ {
				log.Printf("Waiting on lat from peer %d", i)
				total += <-Lat
			}
			log.Printf("succeeded reading %d network updates", *numPeers-1)
			nLat := total / float64(*numPeers-1)

			nW.Write([]string{
				fmt.Sprintf("%.5f", p),
				strconv.Itoa(t),
				fmt.Sprintf("%.5f", nLat),
			})

			//write proportion, time, and value
			lW.Write([]string{
				fmt.Sprintf("%.5f", p),
				strconv.Itoa(t),
				fmt.Sprintf("%.5f", lLat),
			})
			// totalMemUsage := 0
			// for _, p := range peers {
			// 	totalMemUsage += p.Rga.Length()
			// }
			// avgMemUsage := float64(totalMemUsage) / (float64(*numPeers))
			maxUsage := -1
			for _, p := range peers {
				if maxUsage == -1 || maxUsage < p.Rga.Length() {
					maxUsage = p.Rga.Length()
				}
			}

			mW.Write([]string{
				fmt.Sprintf("%.5f", p),
				strconv.Itoa(t),
				strconv.Itoa(maxUsage),
				// fmt.Sprintf("%.5f", avgMemUsage),
			})

			t = t + 1
		}

		time.Sleep(2 * time.Second)

		log.Println("ARE VIEWS EQUAL ? :", peers[0].Rga.Doc.View() == peers[1].Rga.Doc.View())
		log.Println("View 1", peers[0].Rga.Doc.View())
		log.Println("View 2", peers[1].Rga.Doc.View())

		// peers[0].Rga.Doc.UpdateView()
		// peers[1].Rga.Doc.UpdateView()

		// log.Println("ARE VIEWS EQUAL ? :", peers[0].Rga.Doc.View() == peers[1].Rga.Doc.View())
		// log.Println("View 1", peers[0].Rga.Doc.View())
		// log.Println("View 2", peers[1].Rga.Doc.View())
	}

	// flush writes to disk
	nW.Flush()
	lW.Flush()
	mW.Flush()

	if err := nW.Error(); err != nil {
		log.Fatal("network lat csv", err)
	}
	if err := lW.Error(); err != nil {
		log.Fatal("local lat csv", err)
	}
	if err := mW.Error(); err != nil {
		log.Fatal("memory csv", err)
	}
}

func setupPeers(numPeers int, lat chan<- float64) {
	// clean out old peers
	for _, p := range peers {
		if p != nil {
			p.Disconnect()
			p.Shutdown()
		}
	}

	// make peers
	peers = make([]*network.Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		config.Peer = i
		peers[i] = network.MakePeer(config)
		peers[i].InitPeer(nil)
	}

	// wait to set channels to avoid connection handlers
	time.Sleep(1 * time.Second)
	for _, p := range peers {
		p.Lat = lat
	}
}

func noError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

func logError(e error) {
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
	}
}
