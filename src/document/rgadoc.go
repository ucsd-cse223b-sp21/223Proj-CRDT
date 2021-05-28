package document

import (
	"crdt"
	"errors"
)

type RgaDoc struct {
	content string
	cursor  uint
	rgaList []*crdt.RGA
	r       crdt.RGA
}

func (d RgaDoc) View() string {
	return d.content
}

func (d RgaDoc) Append(after int, val string) (Document, error) {
	if after > len(d.content) || after < 0 {
		return nil, errors.New("cannot append outside of doc")
	}
	c := d.content
	c = c[:after] + val + c[after:]
	return RgaDoc{c}, nil
}

func (d RgaDoc) Remove(at int) (Document, error) {
	if at > len(d.content) || at < 0 {
		return nil, errors.New("cannot remove non-existent character")
	}
	c := d.content
	c = c[:at] + c[at+1:]
	return RgaDoc{c}, nil
}

var _ Document = new(RgaDoc)
