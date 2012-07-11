package argo

import (
	"compress/zlib"
	"encoding/binary"
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

func writeNum(w io.Writer, n int) (err error) {
	bytes := []byte{}

	for n >= 0x80 {
		bytes = append(bytes, 0x80|uint8(n&0x7F))
		n = n >> 7
	}

	bytes = append(bytes, uint8(n))

	return binary.Write(w, binary.BigEndian, bytes)
}

func encodePacket(w io.Writer, id int, values ...string) (err error) {
	err = writeNum(w, id)
	if err != nil {
		return err
	}

	for _, value := range values {
		err = writeNum(w, len(value))
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(value))
		if err != nil {
			return err
		}
	}

	return err
}

func encodeURI(w io.Writer, term *Resource, prefixesPtr *[]string) (err error) {
	prefixes := *prefixesPtr
	base, name := SplitPrefix(term.URI)

	var prefixID int = -1

	for i, p := range prefixes {
		if p == base {
			prefixID = i
		}
	}

	if prefixID < 0 {
		prefixID = len(prefixes)
		*prefixesPtr = append(prefixes, base)

		err = encodePacket(w, RDFZ_NEW_PREFIX, base)
		if err != nil {
			return err
		}
	}

	return encodePacket(w, RDFZ_PREFIX_START+prefixID, name)
}

func encodeBNode(w io.Writer, term *BlankNode) (err error) {
	return encodePacket(w, RDFZ_BNODE, term.ID)
}

func encodeLiteral(w io.Writer, term *Literal, prefixesPtr *[]string) (err error) {
	if term.Language != "" {
		return encodePacket(w, RDFZ_LITLANG, term.Value, term.Language)

	} else if term.Datatype != nil {
		err = encodePacket(w, RDFZ_LITDT, term.Value)
		if err != nil {
			return err
		}

		return encodeURI(w, term.Datatype.(*Resource), prefixesPtr)

	}

	return encodePacket(w, RDFZ_LIT, term.Value)
}

func SerializeRDFZ(w io.Writer, tripleChan chan *Triple, errChan chan error, _ map[string]string) {
	defer close(errChan)
	var err error

	z := zlib.NewWriter(w)
	defer z.Close()

	prefixes := make([]string, 0)

	for triple := range tripleChan {
		switch subj := triple.Subject.(type) {
		case *Resource:
			err = encodeURI(z, subj, &prefixes)
		case *BlankNode:
			err = encodeBNode(z, subj)
		}

		if err != nil {
			errChan <- err
			return
		}

		err = encodeURI(z, triple.Predicate.(*Resource), &prefixes)
		if err != nil {
			errChan <- err
			return
		}

		switch obj := triple.Object.(type) {
		case *Resource:
			err = encodeURI(z, obj, &prefixes)
		case *BlankNode:
			err = encodeBNode(z, obj)
		case *Literal:
			err = encodeLiteral(z, obj, &prefixes)
		}

		if err != nil {
			errChan <- err
			return
		}
	}

	err = encodePacket(z, RDFZ_EOF)
	if err != nil {
		errChan <- err
	}
}
