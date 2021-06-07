package crdt

import (
	"errors"
)

type Document interface {
	View() string

	Append(after int, val byte) error

	Remove(at int) error

	UpdateView()
}

var _ Document = new(RgaDoc)

func newRgaDoc(rga RGA) Document {
	return &RgaDoc{
		content: "",
		idList:  make([]Id, 0),
		r:       rga,
	}
}

func NewRgaDoc(r *RGA) Document {
	idList := make([]Id, 1)
	idList[0] = r.Head.Elem.ID

	doc := RgaDoc{"", idList, *r}
	return &doc
}

type RgaDoc struct {
	content string
	idList  []Id
	r       RGA
}

func (d *RgaDoc) View() string {
	return d.content
}

func (d *RgaDoc) Append(after int, val byte) error {
	if after < 0 || after >= len(d.idList) {
		return errors.New("after out of range")
	}

	var afterId Id
	if after == 0 {
		afterId = Id{}
	} else {
		afterId = d.idList[after]
	}

	elem, err := d.r.Append(val, afterId)
	if err != nil {
		return err
	}

	if after == (len(d.content) - 1) {
		d.content = d.content[:after] + string(val)
		d.idList = append(d.idList[:after], elem.ID)
	} else {
		d.content = d.content[:after] + string(val) + d.content[after:]
		d.idList = append(append(d.idList[:after], elem.ID), d.idList[after:]...)
	}

	return nil
}

func (d *RgaDoc) Remove(at int) error {
	if at < 0 || at > len(d.idList) {
		return errors.New("after out of range")
	}

	id := d.idList[at]

	_, err := d.r.Remove(id)
	if err != nil {
		return err
	}

	if at == (len(d.content) - 1) {
		d.content = d.content[:at]
		d.idList = d.idList[:at]
	} else {
		d.content = d.content[:at] + d.content[at+1:]
		d.idList = append(d.idList[:at], d.idList[at+1:]...)
	}

	return nil
}

func (d *RgaDoc) UpdateView() {
	d.content, d.idList = d.r.GetView()
}
