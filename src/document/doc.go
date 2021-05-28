package document

import (
	"crdt"
	"errors"
)

type Document interface {
	View() string

	Append(after int, val string) (Document, error)

	Remove(at int) (Document, error)
}

type NiaveDoc struct {
	content string
}

func (d NiaveDoc) View() string {
	return d.content
}

func (d NiaveDoc) Append(after int, val string) (Document, error) {
	if after > len(d.content) || after < 0 {
		return nil, errors.New("cannot append outside of doc")
	}
	c := d.content
	c = c[:after] + val + c[after:]
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

type RgaDoc struct {
	content string
	idList  []crdt.Id
	rgaList []*crdt.RGA
	r       crdt.RGA
}

func (d RgaDoc) View() string {
	return d.r.getString()
}

func (d RgaDoc) Append(after int, val byte) (Document, error) {
	if after > len(d.content) || after < 0 {
		return nil, errors.New("cannot append outside of doc")
	}
	c := d.content
	c = c[:after] + val + c[after:]
	d.AppendAndUpate(val)
	d.idList =  
	return RgaDoc{c, d.rgaList, d.r}, nil
}

func (d RgaDoc) Remove(at int) (Document, error) {
	if at > len(d.content) || at < 0 {
		return nil, errors.New("cannot remove non-existent character")
	}
	c := d.content
	c = c[:at] + c[at+1:]
	return RgaDoc{c, d.rgaList, d.r}, nil
}

func (d RgaDoc) UpdateAllOtherPeer(elem crdt.Elem) error {
	for i, r := range d.rgaList {
		if i == d.r.Peer {
			continue
		}
		err := r.Update(elem)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d RgaDoc) AppendAndUpate(char byte, after crdt.Id) (crdt.Elem, error) {
	elem, err := d.r.Append(char, after)
	if err != nil {
		return crdt.Elem{}, err
	}
	err = d.UpdateAllOtherPeer(elem)
	if err != nil {
		return crdt.Elem{}, err
	}
	return elem, nil
}

var _ Document = new(RgaDoc)
