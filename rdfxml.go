package argo

import (
	"encoding/xml"
	"fmt"
	"io"
)

/*
type xmlDocument struct {
	XMLName      xml.Name         `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# RDF"`
	Descriptions []xmlDescription `xml:",any"`
}

type xmlDescription struct {
	XMLName    xml.Name
	About      string        `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# about,attr"`
	NodeID     string        `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# nodeID,attr"`
	Properties []xmlProperty `xml:",any"`
}

type xmlProperty struct {
	XMLName  xml.Name
	Resource string `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# resource,attr"`
	NodeID   string `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# nodeID,attr"`
	Datatype string `xml:"http://www.w3.org/1999/02/22-rdf-syntax-ns# datatype,attr"`
	Text     string `xml:",chardata"`
}
*/

const (
	stateTop = iota
	stateDescriptions
	stateProperties
	statePropertyValue
)

var (
	RdfNs = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

	RdfRdf         = xml.Name{RdfNs, "RDF"}
	RdfDescription = xml.Name{RdfNs, "Description"}
	RdfAbout       = xml.Name{RdfNs, "about"}
	RdfNodeID      = xml.Name{RdfNs, "nodeID"}
	RdfResource    = xml.Name{RdfNs, "resource"}
	RdfDatatype    = xml.Name{RdfNs, "datatype"}
	RdfParseType   = xml.Name{RdfNs, "parseType"}

	XmlLang = xml.Name{"xml", "lang"}
)

func Name2Term(name xml.Name) (term Term) {
	return NewResource(name.Space + name.Local)
}

func ParseRDFXML(r io.Reader, tripleChan chan *Triple, errChan chan error) {
	defer close(tripleChan)
	defer close(errChan)

	//var err error

	decoder := xml.NewDecoder(r)
	state := stateTop

	var subject, predicate, datatype Term
	var language string

loop:
	for {
		itok, err := decoder.Token()
		if err != nil {
			if err != io.EOF {
				errChan <- err
			}

			break loop
		}

		switch state {
		case stateTop:
			switch tok := itok.(type) {
			case xml.StartElement:
				if tok.Name != RdfRdf {
					errChan <- fmt.Errorf("Syntax error: expected <rdf:RDF>")
					break loop
				}

				state = stateDescriptions
			}

		case stateDescriptions:
			switch tok := itok.(type) {
			case xml.StartElement:
				subject = nil
				extraAttrs := make([]xml.Attr, 0)

				for _, attr := range tok.Attr {
					if attr.Name == RdfAbout {
						subject = NewResource(attr.Value)
					} else if attr.Name == RdfNodeID {
						subject = NewBlankNode(attr.Value)
					} else {
						extraAttrs = append(extraAttrs, attr)
					}
				}

				if subject == nil {
					subject = NewAnonNode()
				}

				if tok.Name != RdfDescription {
					tripleChan <- NewTriple(subject, A, Name2Term(tok.Name))
				}

				for _, attr := range extraAttrs {
					tripleChan <- NewTriple(subject, Name2Term(attr.Name), NewLiteral(attr.Value))
				}

				state = stateProperties

			case xml.EndElement: // Must be the toplevel tag (</rdf:RDF>)
				break loop
			}

		case stateProperties:
			switch tok := itok.(type) {
			case xml.StartElement:
				predicate = Name2Term(tok.Name)
				language = ""
				datatype = nil
				state = statePropertyValue

				for _, attr := range tok.Attr {
					if attr.Name == RdfResource {
						tripleChan <- NewTriple(subject, predicate, NewResource(attr.Value))
						continue loop

					} else if attr.Name == RdfNodeID {
						tripleChan <- NewTriple(subject, predicate, NewBlankNode(attr.Value))
						continue loop

					} else if attr.Name == RdfDatatype {
						datatype = NewResource(attr.Value)

					} else if attr.Name == XmlLang {
						language = attr.Value

					} else {
						errChan <- fmt.Errorf("Invalid attribute on property tag: %s:%s", attr.Name.Space, attr.Name.Local)
						break loop
					}
				}

			case xml.EndElement: // Must be a description tag (</rdf:Description>)
				state = stateDescriptions
			}

		case statePropertyValue:
			switch tok := itok.(type) {
			case xml.CharData:
				tripleChan <- NewTriple(subject, predicate, NewLiteralWithLanguageAndDatatype(string(tok), language, datatype))

			case xml.EndElement: // Must be a property tag (</foaf:name>)
				state = stateProperties
			}
		}
	}
}

func SerializeRDFXML(w io.Writer, tripleChan chan *Triple, errChan chan error) {
	defer close(errChan)

	var err error

	triplesBySubject := make(map[Term][]*Triple)
	types := make(map[Term]Term)

	for triple := range tripleChan {
		if triple.Predicate == A {
			_, alreadySet := types[triple.Subject]
			_, isResource := triple.Object.(*Resource)

			if !alreadySet && isResource {
				types[triple.Subject] = triple.Object
				continue
			}
		}

		triplesBySubject[triple.Subject] = append(triplesBySubject[triple.Subject], triple)
	}

	_, err = fmt.Fprintf(w, "<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>\n")
	if err != nil {
		errChan <- err
		return
	}

	for subject, triples := range triplesBySubject {
		t, hasType := types[subject]
		subjResource, isResource := subject.(*Resource)
		subjNode, _ := subject.(*BlankNode)

		var subjStr string

		if isResource {
			subjStr = fmt.Sprintf("rdf:about='%s'", subjResource.URI)
		} else {
			subjStr = fmt.Sprintf("rdf:nodeID='%s'", subjNode.ID)
		}

		var tbase, tname string

		if hasType {
			tbase, tname = SplitPrefix(t.(*Resource).URI)
			_, err = fmt.Fprintf(w, "  <t:%s xmlns:t='%s' %s>\n", tname, tbase, subjStr)

		} else {
			_, err = fmt.Fprintf(w, "  <rdf:Description %s>\n", subjStr)
		}

		if err != nil {
			errChan <- err
			continue
		}

		for _, triple := range triples {
			pbase, pname := SplitPrefix(triple.Predicate.(*Resource).URI)

			_, err = fmt.Fprintf(w, "    <p:%s xmlns:p='%s'", pname, pbase)
			if err != nil {
				errChan <- err
				continue
			}

			objResource, isResource := triple.Object.(*Resource)
			objLiteral, isLiteral := triple.Object.(*Literal)
			objNode, _ := triple.Object.(*BlankNode)

			if isResource {
				_, err = fmt.Fprintf(w, " rdf:resource='%s' />\n", objResource.URI)

			} else if isLiteral {
				if objLiteral.Language != "" {
					_, err = fmt.Fprintf(w, " xml:lang='%s'", objLiteral.Language)
					if err != nil {
						errChan <- err
						continue
					}

				} else if objLiteral.Datatype != nil {
					_, err = fmt.Fprintf(w, " rdf:datatype='%s'", objLiteral.Datatype.(*Resource).URI)
					if err != nil {
						errChan <- err
						continue
					}
				}

				_, err = fmt.Fprintf(w, ">%s</p:%s>\n", objLiteral.Value, pname)

			} else {
				_, err = fmt.Fprintf(w, " rdf:nodeID='%s' />\n", objNode.ID)
			}

			if err != nil {
				errChan <- err
				continue
			}
		}

		if hasType {
			_, err = fmt.Fprintf(w, "  </t:%s>\n", tname)
		} else {
			_, err = fmt.Fprintf(w, "  </rdf:Description>\n")
		}

		if err != nil {
			errChan <- err
			continue
		}
	}

	_, err = fmt.Fprintf(w, "</rdf:RDF>\n")
	if err != nil {
		errChan <- err
	}
}

func (graph *Graph) SerializePrettyRDFXML(w io.Writer) (err error) {
	triplesBySubject := make(map[Term][]*Triple)
	types := make(map[Term]Term)

	for triple := range graph.IterTriples() {
		if triple.Predicate == A {
			_, alreadySet := types[triple.Subject]
			_, isResource := triple.Object.(*Resource)

			if !alreadySet && isResource {
				types[triple.Subject] = triple.Object
				continue
			}
		}

		triplesBySubject[triple.Subject] = append(triplesBySubject[triple.Subject], triple)
	}

	_, err = fmt.Fprintf(w, "<rdf:RDF\n  xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'\n")
	if err != nil {
		return err
	}

	for uri, prefix := range graph.Prefixes {
		if prefix != "rdf" {
			_, err = fmt.Fprintf(w, "  xmlns:%s='%s'\n", prefix, uri)
		}
	}

	_, err = fmt.Fprintf(w, ">\n")
	if err != nil {
		return err
	}

	for subject, triples := range triplesBySubject {
		t, hasType := types[subject]
		subjResource, isResource := subject.(*Resource)
		subjNode, _ := subject.(*BlankNode)

		var subjStr string

		if isResource {
			subjStr = fmt.Sprintf("rdf:about='%s'", subjResource.URI)
		} else {
			subjStr = fmt.Sprintf("rdf:nodeID='%s'", subjNode.ID)
		}

		var tbase, tname, tprefix string
		var thasPrefix bool

		if hasType {
			tbase, tname = SplitPrefix(t.(*Resource).URI)
			tprefix, thasPrefix = graph.Prefixes[tbase]

			if thasPrefix {
				_, err = fmt.Fprintf(w, "  <%s:%s %s>\n", tprefix, tname, subjStr)

			} else {
				_, err = fmt.Fprintf(w, "  <t:%s xmlns:t='%s' %s>\n", tname, tbase, subjStr)
			}

		} else {
			_, err = fmt.Fprintf(w, "  <rdf:Description %s>\n", subjStr)
		}

		if err != nil {
			return err
		}

		for _, triple := range triples {
			pbase, pname := SplitPrefix(triple.Predicate.(*Resource).URI)
			pprefix, phasPrefix := graph.Prefixes[pbase]
			//fmt.Println(pbase, pname, pprefix, ok, graph.Prefixes)
			if phasPrefix {
				_, err = fmt.Fprintf(w, "    <%s:%s", pprefix, pname)

			} else {
				_, err = fmt.Fprintf(w, "    <p:%s xmlns:p='%s'", pname, pbase)
			}

			if err != nil {
				return err
			}

			objResource, isResource := triple.Object.(*Resource)
			objLiteral, isLiteral := triple.Object.(*Literal)
			objNode, _ := triple.Object.(*BlankNode)

			if isResource {
				_, err = fmt.Fprintf(w, " rdf:resource='%s' />\n", objResource.URI)

			} else if isLiteral {
				if objLiteral.Language != "" {
					_, err = fmt.Fprintf(w, " xml:lang='%s'", objLiteral.Language)
					if err != nil {
						return err
					}

				} else if objLiteral.Datatype != nil {
					_, err = fmt.Fprintf(w, " rdf:datatype='%s'", objLiteral.Datatype.(*Resource).URI)
					if err != nil {
						return err
					}
				}

				if phasPrefix {
					_, err = fmt.Fprintf(w, ">%s</%s:%s>\n", objLiteral.Value, pprefix, pname)
				} else {
					_, err = fmt.Fprintf(w, ">%s</p:%s>\n", objLiteral.Value, pname)
				}

			} else {
				_, err = fmt.Fprintf(w, " rdf:nodeID='%s' />\n", objNode.ID)
			}

			if err != nil {
				return err
			}
		}

		if hasType {
			if thasPrefix {
				_, err = fmt.Fprintf(w, "  </%s:%s>\n", tprefix, tname)
			} else {
				_, err = fmt.Fprintf(w, "  </t:%s>\n", tname)
			}

		} else {
			_, err = fmt.Fprintf(w, "  </rdf:Description>\n")
		}

		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "</rdf:RDF>\n")
	if err != nil {
		return err
	}

	return nil
}
