package crdt

import (
	"errors"

	"github.com/gorilla/websocket"
)

type Document interface {
	View() string

	Length() int

	Append(after int, val byte) error

	Remove(at int) error

	ComputeView()
	RemoveFromView(e Elem)
	AddToView(e Elem, hint Id)

	AddFront(*websocket.Conn) // maps are pointer types
}

var _ Document = new(RgaDoc)

func NewRgaDoc(r *RGA) Document {
	doc := RgaDoc{r: r}
	doc.ComputeView()
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

	// log.Printf("length of content is %d, length of idList is %d", len(d.content), len(d.idList))
	afterId := d.idList[after]

	// log.Println("before rga append call")

	// elem, err := d.r.Append(val, afterId)
	_, err := d.r.Append(val, afterId)
	return err
}

// remove at should be in range (0,length] (exclusive to 0 / head)
func (d *RgaDoc) Remove(at int) error {
	if at <= 0 || at >= len(d.content) {
		return errors.New("after out of range")
	}

	// log.Println("BEFORE idlist length ", len(d.idList))
	// log.Println("idlist :", d.idList)
	id := d.idList[at]
	_, err := d.r.Remove(id)
	// log.Println("AFTER idlist length ", len(d.idList))
	// log.Println("idlist :", d.idList)

	return err
}

func (d *RgaDoc) ComputeView() {
	d.content, d.idList = d.r.GetView()
	if d.front != nil && d.front.WriteMessage(websocket.TextMessage, []byte(d.content)) != nil {
		d.front = nil
	}
}

func (d *RgaDoc) RemoveFromView(e Elem) {
	for at, id := range d.idList {
		if id == e.ID {
			if at == len(d.content)-1 {
				d.content = d.content[:at]
				d.idList = d.idList[:at]
			} else {
				d.content = d.content[:at] + d.content[at+1:]
				d.idList = append(d.idList[:at], d.idList[at+1:]...)
			}
			break
		}
	}
}
func (d *RgaDoc) AddToView(e Elem, hint Id) {
	for after, id := range d.idList {
		if id == hint {
			if after == len(d.content)-1 {
				d.content = d.content + string(e.Val)
				d.idList = append(d.idList, e.ID)
			} else {
				d.content = d.content[:after+1] + string(e.Val) + d.content[after+1:]
				d.idList = append(d.idList[:after+2], d.idList[after+1:]...)
				d.idList[after+1] = e.ID
			}
			break
		}
	}
}
