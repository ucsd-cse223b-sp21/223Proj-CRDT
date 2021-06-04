package document

import (
	"log"
	"runtime/debug"
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

// func TestDoc(t *testing.T) {
// 	log.Println("testing document")

// 	// create doc
// 	doc_pointer := new(NiaveDoc)
// 	var doc Document = *doc_pointer
// 	as(doc.View() == "")

// 	//append out of range test
// 	_, err := doc.Append(-1, byte('n'))
// 	er(err)
// 	_, err = doc.Append(len(doc.View())+1, byte('n'))
// 	er(err)

// 	// append view test
// 	cursor := 0
// 	for _, cha := range "Hello World!" {
// 		newdoc, err := doc.Append(cursor, byte(cha))
// 		doc = newdoc
// 		cursor++
// 		ne(err)
// 	}
// 	log.Println(doc.View())
// 	as(doc.View() == "Hello World!")

// 	// remove out of range test
// 	_, err = doc.Remove(-1)
// 	er(err)
// 	_, err = doc.Remove(len("Hello World!") + 1)
// 	er(err)

// 	// remove view test
// 	doc, err = doc.Remove(len("Hello World!") - 1)
// 	ne(err)
// 	as(doc.View() == "Hello World")

// 	doc, err = doc.Remove(0)
// 	ne(err)
// 	doc, err = doc.Remove(4)
// 	ne(err)
// 	as(doc.View() == "elloWorld")

// }
