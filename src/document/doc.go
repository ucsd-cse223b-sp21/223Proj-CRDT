package document

import "errors"

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
	c = c[:after+1] + val + c[after+1:]
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
