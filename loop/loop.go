package loop

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/kierdavis/argo"
	"strconv"
	"strings"
)

var XSD = argo.XSD
var LOOPBase = "http://kierdavis.com/data/vocab/loop/"
var LOOP = argo.NewNamespace(LOOPBase)

var Debug = false

func term2str(term argo.Term) (uri string) {
	res, isRes := term.(*argo.Resource)
	if isRes {
		return res.URI
	}
	return "_:" + term.(*argo.BlankNode).ID
}

type Resource argo.Term

type Interpreter struct {
	*argo.Graph
}

func NewInterpreter() (graph *Interpreter) {
	return &Interpreter{argo.NewGraph(argo.NewListStore())}
}

func (graph *Interpreter) FetchIfNeeded(term argo.Term) {
	res, isRes := term.(*argo.Resource)
	if isRes && !graph.HasSubject(term) {
		if Debug {
			fmt.Printf("Fetching '%s'\n", res.URI)
		}

		graph.ParseHTTP(argo.ParseRDFXML, res.URI, "application/rdf+xml")
	}
}

func (graph *Interpreter) Evaluate(term argo.Term, ctx map[string]interface{}) (value interface{}, err error) {
	if ctx == nil {
		ctx = make(map[string]interface{})
	}

	lit, isLit := term.(*argo.Literal)
	if isLit {
		switch lit.Datatype {
		case XSD.Get("boolean"):
			switch lit.Value {
			case "true", "1":
				return true, nil

			case "false", "0":
				return false, nil

			default:
				return nil, fmt.Errorf("Invalid boolean value: %s", lit.Value)
			}

		case XSD.Get("base64Binary"):
			return base64.StdEncoding.DecodeString(lit.Value)

		case XSD.Get("hexBinary"):
			return hex.DecodeString(lit.Value)

		case XSD.Get("float"), XSD.Get("decimal"), XSD.Get("double"):
			return strconv.ParseFloat(lit.Value, 64)

		case XSD.Get("integer"), XSD.Get("nonPositiveInteger"), XSD.Get("negativeInteger"), XSD.Get("long"), XSD.Get("int"), XSD.Get("short"), XSD.Get("byte"), XSD.Get("nonNegativeInteger"), XSD.Get("unsignedLong"), XSD.Get("unsignedInt"), XSD.Get("unsignedShort"), XSD.Get("unsignedByte"), XSD.Get("positiveInteger"):
			return strconv.ParseInt(lit.Value, 10, 64)

		case XSD.Get("anyURI"):
			return Resource(argo.NewResource(lit.Value)), nil

		case XSD.Get("QName"):
			colonPos := strings.Index(lit.Value, ":")
			if colonPos < 0 {
				return nil, fmt.Errorf("No colon found in QName value: %s", lit.Value)
			}

			b := lit.Value[colonPos+1:]
			a, ok := graph.Prefixes[lit.Value[:colonPos]]

			if !ok {
				return nil, fmt.Errorf("Namespace identifier not found in graph prefix map when parsing QName: %s", lit.Value)
			}

			return Resource(argo.NewResource(a + b)), nil

		default:
			return lit.Value, nil
		}
	}

	graph.FetchIfNeeded(term)
	t := graph.Get(term, argo.A)

	switch t {
	case LOOP.Get("Variable"), LOOP.Get("Argument"):
		uri := term2str(term)
		value, ok := ctx[uri]
		if !ok {
			return nil, fmt.Errorf("Reference to unset variable: %s", uri)
		}

		return value, nil
	}

	graph.FetchIfNeeded(t)

	if graph.Get(t, argo.A) == LOOP.Get("Function") {
		uri := term2str(t)

		builtin, ok := builtins[uri]
		if ok {
			valargs := make([]interface{}, len(builtin.ValArgs))
			refargs := make([]Resource, len(builtin.RefArgs))

			for i, argURI := range builtin.ValArgs {
				v, err := graph.Evaluate(graph.MustGet(term, argo.NewResource(argURI)), ctx)
				if err != nil {
					return nil, err
				}

				valargs[i] = v
			}

			for i, argURI := range builtin.RefArgs {
				refargs[i] = Resource(graph.MustGet(term, argo.NewResource(argURI)))
			}

			return builtin.Func(valargs, refargs)

		} else {
			subctx := make(map[string]interface{})

			for triple := range graph.Filter(term, nil, nil) {
				arg := triple.Predicate
				argURI := term2str(arg)
				graph.FetchIfNeeded(arg)

				byRef := false

				obj := graph.Get(arg, LOOP.Get("byReference"))
				if obj != nil {
					lit, isLit = obj.(*argo.Literal)
					byRef = isLit && (lit.Value == "true" || lit.Value == "1")
				}

				if byRef {
					subctx[argURI] = Resource(triple.Object)

				} else {
					v, err := graph.Evaluate(triple.Object, ctx)
					if err != nil {
						return nil, err
					}

					subctx[argURI] = v
				}
			}

			code := graph.Get(t, LOOP.Get("code"))
			if code != nil && code != argo.RDF.Get("nil") {
				for expr := range graph.IterList(code) {
					value, err = graph.Evaluate(expr, subctx)
					if err != nil {
						return nil, err
					}
				}

				return value, nil

			} else {
				return nil, nil
			}
		}
	}

	return Resource(term), nil
}
