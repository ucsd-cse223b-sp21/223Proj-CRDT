package document

import (
	"errors"
	"log"
	"proj/crdt"
)

type Document interface {
	View() string

	Append(after int, val byte) (Document, error)

	Remove(at int) (Document, error)
}

type NiaveDoc struct {
	content string
}

func (d NiaveDoc) View() string {
	return d.content
}

func (d NiaveDoc) Append(after int, val byte) (Document, error) {
	if after > len(d.content) || after < 0 {
		return nil, errors.New("cannot append outside of doc")
	}
	c := d.content
	c = c[:after] + string(rune(val)) + c[after:]
	return NiaveDoc{c}, nil
}

func (d NiaveDoc) Remove(at int) (Document, error) {
	if at > len(d.content) || at < 0 {
		return nil, errors.New("cannot remove non-existent character")
	}
	c := d.content
	c = c[:at] + c[at+1:]
	return NiaveDoc{c}, nil
}

var _ Document = new(NiaveDoc)

func NewRgaDoc(r *crdt.RGA) *RgaDoc {
	idList := make([]crdt.Id, 1)
	idList[0] = r.Head.Elem.ID

	doc := RgaDoc{"", idList, *r}
	return &doc
}

type RgaDoc struct {
	content string
	idList  []crdt.Id
	r       crdt.RGA
}

func (d RgaDoc) View() string {
	text, idL := d.r.GetView()
	d.idList = idL
	return text
}

func (d RgaDoc) Append(after int, val byte) (RgaDoc, error) {
	if after > len(d.content) || after < 0 {
		return RgaDoc{}, errors.New("cannot append outside of doc")
	}

	elem, err := d.AppendToRga(val, d.idList[after])
	if err != nil {
		return RgaDoc{}, err
	}

	tempIdList := append(d.idList[:after], elem.ID)
	tempIdList = append(tempIdList, d.idList[after:]...)

	c := d.content
	c = c[:after] + string(rune(val)) + c[after:]

	return RgaDoc{c, tempIdList, d.r}, nil
}

func (d RgaDoc) Remove(at int) (RgaDoc, error) {
	if at > len(d.content) || at < 0 {
		return RgaDoc{}, errors.New("cannot remove non-existent character")
	}

	err := d.RemoveToRga(d.idList[at])
	if err != nil {
		return RgaDoc{}, err
	}

	c := d.content
	c = c[:at] + c[at+1:]
	return RgaDoc{c, d.idList, d.r}, nil
}

func (d RgaDoc) AppendToRga(char byte, after crdt.Id) (crdt.Elem, error) {
	log.Println("after", after)
	elem, err := d.r.Append(char, after)
	if err != nil {
		return crdt.Elem{}, err
	}
	return elem, nil
}

func (d RgaDoc) RemoveToRga(id crdt.Id) error {
	_, err := d.r.Remove(id)
	if err != nil {
		return err
	}
	return nil
}
