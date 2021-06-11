package crdt

import (
	"errors"
	"log"

	"github.com/gorilla/websocket"
)

type Document interface {
	View() string

	Length() int

	Append(after int, val byte) error

	Remove(at int) error

	UpdateView()

	AddFront(*websocket.Conn) // maps are pointer types
}

var _ Document = new(RgaDoc)

func NewRgaDoc(r *RGA) Document {
	doc := RgaDoc{r: r}
	doc.UpdateView()
	return &doc
}

func (d *RgaDoc) AddFront(front *websocket.Conn) {
	d.front = front
}

type RgaDoc struct {
	content string
	idList  []Id
	r       *RGA
	front   *websocket.Conn
}

func (d *RgaDoc) View() string {
	return d.content[1:] // ignore head
}

func (d *RgaDoc) Length() int {
	return len(d.content) - 1
}

// after is in range [0,length]
func (d *RgaDoc) Append(after int, val byte) error {
	if after < 0 || after >= len(d.content) {
		return errors.New("after out of range")
	}

	log.Printf("length of content is %d, length of idList is %d", len(d.content), len(d.idList))
	afterId := d.idList[after]

	log.Println("before rga append call")

	// elem, err := d.r.Append(val, afterId)
	_, err := d.r.Append(val, afterId)
	if err != nil {
		return err
	}

	// log.Println("after rga append call")

	// if after == len(d.content)-1 {
	// 	d.content = d.content + string(val)
	// 	d.idList = append(d.idList, elem.ID)
	// } else {
	// 	d.content = d.content[:after+1] + string(val) + d.content[after+1:]
	// 	d.idList = append(append(d.idList[:after+1], elem.ID), d.idList[after+1:]...)
	// }

	return nil
}

// remove at should be in range (0,length] (exclusive to 0 / head)
func (d *RgaDoc) Remove(at int) error {
	if at <= 0 || at >= len(d.content) {
		return errors.New("after out of range")
	}

	id := d.idList[at]

	_, err := d.r.Remove(id)
	if err != nil {
		return err
	}

	// if at == len(d.content)-1 {
	// 	d.content = d.content[:at]
	// 	d.idList = d.idList[:at]
	// } else {
	// 	d.content = d.content[:at] + d.content[at+1:]
	// 	d.idList = append(d.idList[:at], d.idList[at+1:]...)
	// }

	return nil
}

func (d *RgaDoc) UpdateView() {
	d.content, d.idList = d.r.GetView()
	if d.front != nil && d.front.WriteMessage(websocket.TextMessage, []byte(d.content)) != nil {
		d.front = nil
	}
}
