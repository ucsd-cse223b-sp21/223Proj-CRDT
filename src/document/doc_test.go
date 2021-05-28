package document

import (
	"log"
	"testing"
)

func ne(e error) {
	if e != nil {
		log.Fatal(e)
	}
}
func er(e error) {
	if e == nil {
		log.Fatal("didn't get an error, when it should")
	}
}
func as(cond bool) {
	if !cond {
		log.Fatal("assertion failed")
	}
}

func TestDoc(t *testing.T) {
	log.Println("testing document")

	// create doc
	doc := new(NiaveDoc)
	as(doc.content == "")

	//append out of range test
	_, err := doc.Append(-1, "nnnnnnnnnn")
	er(err)
	_, err = doc.Append(len(doc.content)+1, "nnnnnnnnnn")
	er(err)

	// append view test
	newdoc, err := doc.Append(0, "Hello")
	ne(err)
	as(newdoc.View() == "Hello")
	newdoc, err = newdoc.Append(len(newdoc.View()), " ")
	ne(err)
	as(newdoc.View() == "Hello ")
	newdoc, err = newdoc.Append(len(newdoc.View()), "World")
	ne(err)
	as(newdoc.View() == "Hello World")
	newdoc, err = newdoc.Append(len(newdoc.View()), "!")
	ne(err)
	as(newdoc.View() == "Hello World!")

	// remove out of range test
	_, err = newdoc.Remove(-1)
	er(err)
	_, err = newdoc.Remove(len("Hello World!") + 1)
	er(err)

	// remove view test
	newdoc, err = newdoc.Remove(len("Hello World!") - 1)
	ne(err)
	as(newdoc.View() == "Hello World")

	newdoc, err = newdoc.Remove(0)
	ne(err)
	newdoc, err = newdoc.Remove(4)
	ne(err)
	as(newdoc.View() == "elloWorld")

}
