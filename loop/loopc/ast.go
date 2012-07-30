package main

import (
	"encoding/base64"
	"fmt"
	"github.com/kierdavis/argo"
	"strings"
)

func str2term(uri string) (term argo.Term) {
	if len(uri) >= 2 && uri[0] == '_' && uri[1] == ':' {
		return argo.NewBlankNode(uri[2:])
	}
	return argo.NewResource(uri)
}

type Func struct {
	URI  string
	Args []*Arg
	Body []Statement
}

func (f *Func) ToLOOP() (code string) {
	argStrs := make([]string, len(f.Args))

	for i, arg := range f.Args {
		argStrs[i] = arg.ToLOOP()
	}

	code = fmt.Sprintf("func <%s>(%s) {", f.URI, strings.Join(argStrs, ", "))

	for _, stmt := range f.Body {
		code += "\n  " + stmt.ToLOOP()
	}

	return code + "\n}"
}

func (f *Func) ToRDF(graph *argo.Graph) (term argo.Term) {
	term = str2term(f.URI)
	graph.AddTriple(term, argo.A, LOOP.Get("Function"))

	if len(f.Args) > 0 {
		argsTerm := argo.NewAnonNode()
		graph.AddTriple(term, LOOP.Get("args"), argsTerm)
		graph.AddTriple(argsTerm, argo.A, argo.RDF.Get("Bag"))
		argsList := graph.EncodeContainer(argsTerm)

		for _, arg := range f.Args {
			argsList <- arg.ToRDF(graph)
		}

		close(argsList)
	}

	var bodyList chan argo.Term

	for _, stmt := range f.Body {
		po, isPO := stmt.(*PredicateObjectPair)
		if isPO {
			var subject, object argo.Term

			if po.SubjVar == "" {
				subject = term
			} else {
				subject = str2term(po.SubjVar)
			}

			predicate := str2term(po.Predicate)

			res, isRes := po.Object.(ResourceObject)
			if isRes {
				object = str2term(string(res))
			} else {
				lit := po.Object.(*LiteralObject)
				object = argo.NewLiteralWithLanguageAndDatatype(lit.Value, lit.Language, str2term(lit.Datatype))
			}

			graph.AddTriple(subject, predicate, object)

		} else {
			expr := stmt.(Expression)

			if bodyList == nil {
				bodyTerm := argo.NewAnonNode()
				graph.AddTriple(term, LOOP.Get("code"), bodyTerm)
				bodyList = graph.EncodeList(bodyTerm)
			}

			bodyList <- expr.ToRDF(graph)
		}
	}

	if bodyList != nil {
		close(bodyList)
	}

	return term
}

type Arg struct {
	URI   string
	Type  Type
	Label string
}

func (arg *Arg) ToLOOP() (code string) {
	code = fmt.Sprintf("<%s>", arg.URI)

	if arg.Type != nil {
		code += fmt.Sprintf(" %s", arg.Type.ToLOOP())
	}

	if arg.Label != "" {
		code += fmt.Sprintf(" \"%s\"", arg.Label)
	}

	return code
}

func (arg *Arg) ToRDF(graph *argo.Graph) (term argo.Term) {
	term = str2term(arg.URI)
	graph.AddTriple(term, argo.A, LOOP.Get("Argument"))

	if arg.Type != nil {
		graph.AddTriple(term, LOOP.Get("restrictType"), arg.Type.ToRDF(graph))
	}

	if arg.Label != "" {
		graph.AddTriple(term, argo.RDFS.Get("label"), argo.NewLiteral(arg.Label))
	}

	return term
}

type Type interface {
	ToLOOP() string
	ToRDF(*argo.Graph) argo.Term
}

type AtomicType int

func (t AtomicType) ToLOOP() (code string) {
	switch t {
	case Boolean:
		return "boolean"
	case Data:
		return "data"
	case Float:
		return "float"
	case Integer:
		return "integer"
	case Resource:
		return "resource"
	case String:
		return "string"
	}

	return ""
}

func (t AtomicType) ToRDF(graph *argo.Graph) (term argo.Term) {
	switch t {
	case Boolean:
		return LOOP.Get("Boolean")
	case Data:
		return LOOP.Get("Data")
	case Float:
		return LOOP.Get("Float")
	case Integer:
		return LOOP.Get("Integer")
	case Resource:
		return LOOP.Get("Resource")
	case String:
		return LOOP.Get("String")
	}

	return nil
}

const (
	Boolean AtomicType = iota
	Data
	Float
	Integer
	Resource
	String
)

type Statement interface {
	ToLOOP() string
}

type PredicateObjectPair struct {
	SubjVar   string
	Predicate string
	Object    Object
}

func (stmt *PredicateObjectPair) ToLOOP() (code string) {
	if stmt.SubjVar != "" {
		return fmt.Sprintf("<%s> of <%s> %s", stmt.Predicate, stmt.SubjVar, stmt.Object.ToLOOP())
	} else {
		return fmt.Sprintf("<%s> %s", stmt.Predicate, stmt.Object.ToLOOP())
	}

	return ""
}

type Expression interface {
	Statement
	ToRDF(*argo.Graph) argo.Term
}

type FuncCall struct {
	URI  string
	Args map[string]Expression
}

func (expr *FuncCall) ToLOOP() (code string) {
	argStrs := make([]string, len(expr.Args))

	i := 0
	for uri, arg := range expr.Args {
		argStrs[i] = fmt.Sprintf("<%s> %s", uri, arg.ToLOOP())
		i++
	}

	return fmt.Sprintf("<%s>(%s)", expr.URI, strings.Join(argStrs, ", "))
}

func (expr *FuncCall) ToRDF(graph *argo.Graph) (term argo.Term) {
	term = argo.NewAnonNode()

	graph.AddTriple(term, argo.A, str2term(expr.URI))

	for uri, arg := range expr.Args {
		graph.AddTriple(term, str2term(uri), arg.ToRDF(graph))
	}

	return term
}

type BooleanConstant bool

func (expr BooleanConstant) ToLOOP() (code string) {
	if expr {
		return "\"true\"^^xsd:boolean"
	}

	return "\"false\"^^xsd:boolean"
}

func (expr BooleanConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	if expr {
		return argo.NewLiteralWithDatatype("true", argo.XSD.Get("boolean"))
	}

	return argo.NewLiteralWithDatatype("false", argo.XSD.Get("boolean"))
}

type DataConstant []byte

func (expr DataConstant) ToLOOP() (code string) {
	return fmt.Sprintf("\"%s\"^^xsd:base64Binary", base64.StdEncoding.EncodeToString([]byte(expr)))
}

func (expr DataConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	return argo.NewLiteralWithDatatype(base64.StdEncoding.EncodeToString([]byte(expr)), argo.XSD.Get("base64Binary"))
}

type FloatConstant float64

func (expr FloatConstant) ToLOOP() (code string) {
	return fmt.Sprintf("\"%f\"^^xsd:double", float64(expr))
}

func (expr FloatConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	return argo.NewLiteralWithDatatype(fmt.Sprintf("%f", float64(expr)), argo.XSD.Get("double"))
}

type IntegerConstant int64

func (expr IntegerConstant) ToLOOP() (code string) {
	return fmt.Sprintf("\"%d\"^^xsd:integer", int64(expr))
}

func (expr IntegerConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	return argo.NewLiteralWithDatatype(fmt.Sprintf("%d", float64(expr)), argo.XSD.Get("integer"))
}

type ResourceConstant string

func (expr ResourceConstant) ToLOOP() (code string) {
	return fmt.Sprintf("<%s>", string(expr))
}

func (expr ResourceConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	return argo.NewResource(string(expr))
}

type StringConstant string

func (expr StringConstant) ToLOOP() (code string) {
	return fmt.Sprintf("\"%s\"", string(expr))
}

func (expr StringConstant) ToRDF(graph *argo.Graph) (term argo.Term) {
	return argo.NewLiteral(string(expr))
}

type Object interface {
	ToLOOP() string
}

type ResourceObject string

func (obj ResourceObject) ToLOOP() (code string) {
	return fmt.Sprintf("<%s>", string(obj))
}

type LiteralObject struct {
	Value    string
	Language string
	Datatype string
}

func (obj *LiteralObject) ToLOOP() (code string) {
	if obj.Language != "" {
		return fmt.Sprintf("\"%s\"@%s", obj.Value, obj.Language)
	}

	if obj.Datatype != "" {
		return fmt.Sprintf("\"%s\"^^<%s>", obj.Value, obj.Datatype)
	}

	return fmt.Sprintf("\"%s\"", obj.Value)
}
