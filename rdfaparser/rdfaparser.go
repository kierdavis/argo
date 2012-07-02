package rdfaparser

import (
	"bufio"
	"code.google.com/p/go-html-transform/h5"
	"github.com/kierdavis/argo"
	"io"
	"strings"
)

func getAttr(node *h5.Node, name string) (value string, ok bool) {
	for _, attr := range node.Attr {
		if attr.Name == name {
			return attr.Value, true
		}
	}

	return "", false
}

func expandURI(s string, vocabBase string, prefixMap map[string]string) (r string) {
	p := strings.IndexRune(s, ':')
	if p < 0 {
		return vocabBase + s
	}

	a := s[:p]
	b := s[p+1:]
	return prefixMap[a] + b
}

func traverseNode(node *h5.Node, tripleChan chan *argo.Triple, subject argo.Term, vocabBase string, prefixMap map[string]string, nsMap map[string]string) {
	if node.Type == h5.ElementNode {
		for _, attr := range node.Attr {
			if strings.HasPrefix(attr.Name, "xmlns:") {
				prefix := attr.Name[6:]
				uri := attr.Value

				prefixMap[prefix] = uri
				nsMap[uri] = prefix
			}
		}

		newVocabBase, ok := getAttr(node, "vocab")
		if ok {
			vocabBase = newVocabBase
		}

		typeof, ok := getAttr(node, "typeof")
		if ok {
			resource, ok := getAttr(node, "resource")
			if ok {
				subject = argo.NewResource(resource)
			} else {
				subject = argo.NewAnonNode()
			}

			tripleChan <- argo.NewTriple(subject, argo.A, argo.NewResource(expandURI(typeof, vocabBase, prefixMap)))
		}

		property, ok := getAttr(node, "property")
		if ok {
			predicate := argo.NewResource(expandURI(property, vocabBase, prefixMap))

			var value string
			var object argo.Term

			content, ok := getAttr(node, "content")
			if ok {
				value = content

			} else {
				if len(node.Children) == 0 {
					value = ""
				} else {
					value = node.Children[0].Data()
				}
			}

			datatype, ok := getAttr(node, "datatype")
			if ok {
				object = argo.NewLiteralWithDatatype(value, argo.NewResource(datatype))

			} else {
				language, ok := getAttr(node, "xml:lang")
				if ok {
					object = argo.NewLiteralWithLanguage(value, language)
				} else {
					object = argo.NewLiteral(value)
				}
			}

			tripleChan <- argo.NewTriple(subject, predicate, object)
		}

		rel, ok := getAttr(node, "rel")
		if ok {
			predicate := argo.NewResource(expandURI(rel, vocabBase, prefixMap))

			href, ok := getAttr(node, "href")
			if ok {
				tripleChan <- argo.NewTriple(subject, predicate, argo.NewResource(href))
			}
		}

		rev, ok := getAttr(node, "rev")
		if ok {
			predicate := argo.NewResource(expandURI(rev, vocabBase, prefixMap))

			href, ok := getAttr(node, "href")
			if ok {
				tripleChan <- argo.NewTriple(argo.NewResource(href), predicate, subject)
			}
		}
	}

	for _, child := range node.Children {
		traverseNode(child, tripleChan, subject, vocabBase, prefixMap, nsMap)
	}
}

func NewRDFAParser(documentURI string) (p argo.Parser) {
	return func(r io.Reader, tripleChan chan *argo.Triple, errChan chan error, prefixes map[string]string) {
		defer close(tripleChan)
		defer close(errChan)

		br := bufio.NewReader(r)
		bhead, err := br.Peek(256)
		if err != nil {
			errChan <- err
			return
		}

		head := string(bhead)

		if strings.HasPrefix(head, "<?") {
			end := strings.IndexRune(head, '>')
			_, err = br.Read(make([]byte, end+1))
			if err != nil {
				errChan <- err
				return
			}
		}

		p := h5.NewParser(br)
		err = p.Parse()
		if err != nil {
			errChan <- err
			return
		}

		traverseNode(p.Tree(), tripleChan, argo.NewResource(documentURI), "", make(map[string]string), prefixes)
	}
}
