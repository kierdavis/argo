package sparql

import (
	"fmt"
	"github.com/kierdavis/argo"
)

type Element interface {
	String() string
}

type TriplePattern struct {
	Subject   Expression
	Predicate Expression
	Object    Expression
}

func (element *TriplePattern) String() (str string) {
	return fmt.Sprintf("%s %s %s .", element.Subject, element.Predicate, element.Object)
}

func parseSPINElement(graph *argo.Graph, root argo.Term) (element Element, err error) {
	subject := graph.Get(root, SP.Get("subject"))
	object := graph.Get(root, SP.Get("object"))

	if subject != nil && object != nil {
		predicate := graph.Get(root, SP.Get("predicate"))
		//path := graph.Get(root, SP.Get("path"))

		if predicate != nil {
			sexp, err := parseSPINExpression(graph, subject)
			if err != nil {
				return nil, err
			}

			pexp, err := parseSPINExpression(graph, predicate)
			if err != nil {
				return nil, err
			}

			oexp, err := parseSPINExpression(graph, object)
			if err != nil {
				return nil, err
			}

			return &TriplePattern{sexp, pexp, oexp}, nil
		}
	}

	return nil, fmt.Errorf("Invalid element construction")
}
