package sparql

import (
	"fmt"
	"github.com/kierdavis/argo"
	"strings"
)

type SelectQuery struct {
	ResultVariables []Expression
	Where           []Element
}

func ParseSelectQueryFromSPARQL(sparql string) (query *SelectQuery, err error) {
	return nil, fmt.Errorf("not implemented")
}

func ParseSelectQueryFromSPIN(graph *argo.Graph, root argo.Term) (query *SelectQuery, err error) {
	query = new(SelectQuery)

	resultVariables := graph.Get(root, SP.Get("resultVariables"))
	if resultVariables != nil {
		for resultVariable := range graph.IterList(resultVariables) {
			expr, err := parseSPINExpression(graph, resultVariable)
			if err != nil {
				return nil, err
			}

			query.ResultVariables = append(query.ResultVariables, expr)
		}
	}

	wheres := graph.Get(root, SP.Get("where"))
	if wheres != nil {
		for where := range graph.IterList(wheres) {
			element, err := parseSPINElement(graph, where)
			if err != nil {
				return nil, err
			}

			query.Where = append(query.Where, element)
		}
	}

	return query, nil
}

func (query *SelectQuery) String() (str string) {
	var resultVariableString string
	var whereString string

	if query.ResultVariables == nil {
		resultVariableStrings := make([]string, len(query.ResultVariables))

		for i, resultVariable := range query.ResultVariables {
			resultVariableStrings[i] = resultVariable.String()
		}

		resultVariableString = strings.Join(resultVariableStrings, ", ") + " "
	}

	if query.Where != nil {
		whereStrings := make([]string, len(query.Where))

		for i, where := range query.Where {
			whereStrings[i] = where.String()
		}

		whereString = strings.Join(whereStrings, " ")
	}

	return fmt.Sprintf("SELECT %sWHERE {%s}", resultVariableString, whereString)
}
