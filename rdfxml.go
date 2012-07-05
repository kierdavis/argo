/*
	Copyright (c) 2012 Kier Davis

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
	associated documentation files (the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial
	portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
	NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
	NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
	OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package argo

import (
	"encoding/xml"
	"fmt"
	"io"
)

// State IDs for the state machine.
const (
	stateTop = iota
	stateDescriptions
	stateProperties
	statePropertyValue
)

// XML tag & attribute names used by the parser.
var (
	rdfNs = "http://www.w3.org/1999/02/22-rdf-syntax-ns#"

	rdfRdf         = xml.Name{rdfNs, "RDF"}
	rdfDescription = xml.Name{rdfNs, "Description"}
	rdfAbout       = xml.Name{rdfNs, "about"}
	rdfNodeID      = xml.Name{rdfNs, "nodeID"}
	rdfResource    = xml.Name{rdfNs, "resource"}
	rdfDatatype    = xml.Name{rdfNs, "datatype"}
	rdfParseType   = xml.Name{rdfNs, "parseType"}

	xmlLang = xml.Name{"xml", "lang"}
)

// Converts an xml.Name into a IRI reference term (used for predicate and inline type parsing).
func name2Term(name xml.Name) (term Term) {
	return NewResource(name.Space + name.Local)
}

// Function ParseRDFXML parses RDF/XML from r and sends parsed triples on tripleChan and errors on
// errChan. Both channels are closed when execution is done.
func ParseRDFXML(r io.Reader, tripleChan chan *Triple, errChan chan error, prefixes map[string]string) {
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
				if tok.Name != rdfRdf {
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
					if attr.Name == rdfAbout {
						subject = NewResource(attr.Value)
					} else if attr.Name == rdfNodeID {
						subject = NewBlankNode(attr.Value)
					} else {
						extraAttrs = append(extraAttrs, attr)
					}
				}

				if subject == nil {
					subject = NewAnonNode()
				}

				if tok.Name != rdfDescription {
					tripleChan <- NewTriple(subject, A, name2Term(tok.Name))
				}

				for _, attr := range extraAttrs {
					tripleChan <- NewTriple(subject, name2Term(attr.Name), NewLiteral(attr.Value))
				}

				state = stateProperties

			case xml.EndElement: // Must be the toplevel tag (</rdf:RDF>)
				break loop
			}

		case stateProperties:
			switch tok := itok.(type) {
			case xml.StartElement:
				predicate = name2Term(tok.Name)
				language = ""
				datatype = nil
				state = statePropertyValue

				for _, attr := range tok.Attr {
					if attr.Name == rdfResource {
						tripleChan <- NewTriple(subject, predicate, NewResource(attr.Value))
						continue loop

					} else if attr.Name == rdfNodeID {
						tripleChan <- NewTriple(subject, predicate, NewBlankNode(attr.Value))
						continue loop

					} else if attr.Name == rdfDatatype {
						datatype = NewResource(attr.Value)

					} else if attr.Name == xmlLang {
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

// Function SerializeRDFXML writes RDF/XML to w, sourcing triples from tripleChan and sending errors
// to errChan. errChan is closed when execution is done.
func SerializeRDFXML(w io.Writer, tripleChan chan *Triple, errChan chan error, prefixes map[string]string) {
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

	_, err = fmt.Fprintf(w, "<rdf:RDF\n  xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'\n")
	if err != nil {
		errChan <- err
		return
	}

	for uri, prefix := range prefixes {
		if prefix != "rdf" {
			_, err = fmt.Fprintf(w, "  xmlns:%s='%s'\n", prefix, uri)
		}
	}

	_, err = fmt.Fprintf(w, ">\n")
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

		var tbase, tname, tprefix string
		var thasPrefix bool

		if hasType {
			tbase, tname = SplitPrefix(t.(*Resource).URI)
			tprefix, thasPrefix = prefixes[tbase]

			if thasPrefix {
				_, err = fmt.Fprintf(w, "  <%s:%s %s>\n", tprefix, tname, subjStr)

			} else {
				_, err = fmt.Fprintf(w, "  <%s xmlns='%s' %s>\n", tname, tbase, subjStr)
			}

		} else {
			_, err = fmt.Fprintf(w, "  <rdf:Description %s>\n", subjStr)
		}

		if err != nil {
			errChan <- err
			continue
		}

		for _, triple := range triples {
			pbase, pname := SplitPrefix(triple.Predicate.(*Resource).URI)
			pprefix, phasPrefix := prefixes[pbase]
			//fmt.Println(pbase, pname, pprefix, ok, graph.Prefixes)
			if phasPrefix {
				_, err = fmt.Fprintf(w, "    <%s:%s", pprefix, pname)

			} else {
				_, err = fmt.Fprintf(w, "    <%s xmlns='%s'", pname, pbase)
			}

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

				if phasPrefix {
					_, err = fmt.Fprintf(w, ">%s</%s:%s>\n", objLiteral.Value, pprefix, pname)
				} else {
					_, err = fmt.Fprintf(w, ">%s</%s>\n", objLiteral.Value, pname)
				}

			} else {
				_, err = fmt.Fprintf(w, " rdf:nodeID='%s' />\n", objNode.ID)
			}

			if err != nil {
				errChan <- err
				continue
			}
		}

		if hasType {
			if thasPrefix {
				_, err = fmt.Fprintf(w, "  </%s:%s>\n", tprefix, tname)
			} else {
				_, err = fmt.Fprintf(w, "  </%s>\n", tname)
			}

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
