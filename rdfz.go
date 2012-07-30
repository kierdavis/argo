package argo

import (
	"compress/zlib"
	"encoding/binary"
	"github.com/kierdavis/go/splaytree"
	"io"
)

const (
	RDFZ_EOF = iota
	RDFZ_BNODE
	RDFZ_LIT
	RDFZ_LITLANG
	RDFZ_LITDT
	RDFZ_NEW_PREFIX
	RDFZ_PREFIX_START
)

func rdfzWriteNum(w io.Writer, n uint) (err error) {
	bytes := []byte{}

	for n >= 0x80 {
		bytes = append(bytes, 0x80|uint8(n&0x7F))
		n = n >> 7
	}

	bytes = append(bytes, uint8(n))

	return binary.Write(w, binary.BigEndian, bytes)
}

func rdfzReadNum(r io.Reader) (n uint, err error) {
	var b byte

	sh := uint(0)

	for {
		err = binary.Read(r, binary.BigEndian, &b)
		if err != nil {
			return 0, err
		}

		n |= (uint(b&0x7F) << sh)
		sh += 7

		if b&0x80 == 0 {
			return n, nil
		}
	}

	return 0, nil
}

func rdfzWriteString(w io.Writer, s string) (err error) {
	err = rdfzWriteNum(w, uint(len(s)))
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(s))
	return err
}

func rdfzReadString(r io.Reader) (s string, err error) {
	l, err := rdfzReadNum(r)
	if err != nil {
		return "", err
	}

	buffer := make([]byte, l)
	n, err := r.Read(buffer)
	if err != nil {
		return "", err
	}

	s = string(buffer[:n])

	for uint(n) < l {
		m, err := r.Read(buffer[:l-uint(n)])
		if err != nil {
			return "", err
		}

		s += string(buffer[:m])
		n += m
	}

	return s, nil
}

func rdfzRead2Strings(r io.Reader) (s1 string, s2 string, err error) {
	s1, err = rdfzReadString(r)
	if err != nil {
		return "", "", err
	}

	s2, err = rdfzReadString(r)
	if err != nil {
		return "", "", err
	}

	return s1, s2, nil
}

func rdfzEncodePacket(w io.Writer, id uint, values ...string) (err error) {
	err = rdfzWriteNum(w, id)
	if err != nil {
		return err
	}

	for _, value := range values {
		err = rdfzWriteString(w, value)
		if err != nil {
			return err
		}
	}

	return err
}

type rdfzPrefixTable struct {
	Tree  *splaytree.SplayTree
	Count uint
}

func (t *rdfzPrefixTable) Lookup(base string) (id uint, ok bool) {
	i, ok := t.Tree.Search(splaytree.NewHashKey(base))
	if ok {
		return i.(uint), true
	}

	return 0, false
}

func (t *rdfzPrefixTable) Add(base string) (id uint) {
	id = t.Count
	t.Count++
	t.Tree = t.Tree.Insert(splaytree.NewHashKey(base), id)

	return id
}

func rdfzEncodeURI(w io.Writer, term *Resource, prefixes *rdfzPrefixTable) (err error) {
	base, name := SplitPrefix(term.URI)
	prefixID, ok := prefixes.Lookup(base)

	if !ok {
		prefixID = prefixes.Add(base)
		err = rdfzEncodePacket(w, RDFZ_NEW_PREFIX, base)
		if err != nil {
			return err
		}
	}

	return rdfzEncodePacket(w, RDFZ_PREFIX_START+prefixID, name)
}

func rdfzEncodeBNode(w io.Writer, term *BlankNode) (err error) {
	return rdfzEncodePacket(w, RDFZ_BNODE, term.ID)
}

func rdfzEncodeLiteral(w io.Writer, term *Literal, prefixes *rdfzPrefixTable) (err error) {
	if term.Language != "" {
		return rdfzEncodePacket(w, RDFZ_LITLANG, term.Value, term.Language)

	} else if term.Datatype != nil {
		err = rdfzEncodePacket(w, RDFZ_LITDT, term.Value)
		if err != nil {
			return err
		}

		return rdfzEncodeURI(w, term.Datatype.(*Resource), prefixes)

	}

	return rdfzEncodePacket(w, RDFZ_LIT, term.Value)
}

func ParseRDFZ(r io.Reader, tripleChan chan *Triple, errChan chan error, _ map[string]string) {
	defer close(tripleChan)
	defer close(errChan)

	z, err := zlib.NewReader(r)
	if err != nil {
		errChan <- err
		return
	}

	defer z.Close()

	prefixes := make([]string, 0)
	terms := make([]Term, 0, 3)

	var dtlit string

	for {
		packetID, err := rdfzReadNum(z)
		if err != nil {
			errChan <- err
			return
		}

		switch packetID {
		case RDFZ_EOF:
			return

		case RDFZ_BNODE:
			id, err := rdfzReadString(z)
			if err != nil {
				errChan <- err
				return
			}

			terms = append(terms, NewBlankNode(id))

		case RDFZ_LIT:
			value, err := rdfzReadString(z)
			if err != nil {
				errChan <- err
				return
			}

			terms = append(terms, NewLiteral(value))

		case RDFZ_LITLANG:
			value, lang, err := rdfzRead2Strings(z)
			if err != nil {
				errChan <- err
				return
			}

			terms = append(terms, NewLiteralWithLanguage(value, lang))

		case RDFZ_LITDT:
			dtlit, err = rdfzReadString(z)
			if err != nil {
				errChan <- err
				return
			}

		case RDFZ_NEW_PREFIX:
			base, err := rdfzReadString(z)
			if err != nil {
				errChan <- err
				return
			}

			prefixes = append(prefixes, base)

		default:
			prefixID := packetID - RDFZ_PREFIX_START

			if prefixID >= uint(len(prefixes)) {
				continue
			}

			name, err := rdfzReadString(z)
			if err != nil {
				errChan <- err
				return
			}

			term := NewResource(prefixes[prefixID] + name)

			if dtlit == "" {
				terms = append(terms, term)
			} else {
				terms = append(terms, NewLiteralWithDatatype(dtlit, term))
				dtlit = ""
			}
		}

		if len(terms) == 3 {
			tripleChan <- NewTriple(terms[0], terms[1], terms[2])
			terms = terms[:0]
		}
	}
}

func SerializeRDFZ(w io.Writer, tripleChan chan *Triple, errChan chan error, _ map[string]string) {
	defer close(errChan)
	var err error

	z := zlib.NewWriter(w)
	defer z.Close()

	prefixes := new(rdfzPrefixTable)

	for triple := range tripleChan {
		switch subj := triple.Subject.(type) {
		case *Resource:
			err = rdfzEncodeURI(z, subj, prefixes)
		case *BlankNode:
			err = rdfzEncodeBNode(z, subj)
		}

		if err != nil {
			errChan <- err
			return
		}

		err = rdfzEncodeURI(z, triple.Predicate.(*Resource), prefixes)
		if err != nil {
			errChan <- err
			return
		}

		switch obj := triple.Object.(type) {
		case *Resource:
			err = rdfzEncodeURI(z, obj, prefixes)
		case *BlankNode:
			err = rdfzEncodeBNode(z, obj)
		case *Literal:
			err = rdfzEncodeLiteral(z, obj, prefixes)
		}

		if err != nil {
			errChan <- err
			return
		}
	}

	err = rdfzEncodePacket(z, RDFZ_EOF)
	if err != nil {
		errChan <- err
	}
}
