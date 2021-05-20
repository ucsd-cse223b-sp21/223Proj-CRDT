package crdt

import (
	"log"
	"runtime/debug"
	"testing"
)

func ne(e error) {
	if e != nil {
		debug.PrintStack()
		log.Fatal(e)
	}
}
func er(e error) {
	if e == nil {
		debug.PrintStack()
		log.Fatal("didn't get an error, when it should")
	}
}
func as(cond bool) {
	if !cond {
		debug.PrintStack()
		log.Fatal("assertion failed")
	}
}

func TestSingleUser(t *testing.T) {
	// creating new rga
	r := newRGA(1, 1)

	//typing 'A'

	_, err := r.append(byte('A'), r.head.elem.id)
	ne(err)

	//rga should contain 'A'

	log.Println("test")
	log.Println(r.getString())
}
