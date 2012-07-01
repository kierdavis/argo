package sparql

import (
	"fmt"
	"github.com/kierdavis/argo"
)

type Expression interface {
	String() string
}

type Variable string

func (expr Variable) String() (str string) {
	return fmt.Sprintf("?%s", expr)
}

func parseSPINExpression(graph *argo.Graph, root argo.Term) (expr Expression, err error) {
	varName := graph.Get(root, SP.Get("varName"))
	if varName != nil {
		varNameLit, ok := varName.(*argo.Literal)
		if !ok {
			return nil, fmt.Errorf("sp:varName should be a literal")
		}

		return Variable(varNameLit.Value), nil
	}

	return root, nil // Term implements Expression and its String() returns an NTriples-like representation
}
