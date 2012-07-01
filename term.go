package argo

import (
	"fmt"
	"github.com/kierdavis/goutil"
)

// Generate thread-safe unique IDs

var newNodeIDs = util.UniqueInts()

type Term interface {
	String() string
	Equal(other Term) bool
}

type Resource struct {
	URI string
}

func NewResource(uri string) (term Term) {
	return Term(&Resource{URI: uri})
}

func (term Resource) String() (str string) {
	return fmt.Sprintf("<%s>", term.URI)
}

func (term Resource) Equal(other Term) bool {
	if spec, ok := other.(*Resource); ok {
		return term.URI == spec.URI
	}

	return false
}

type Literal struct {
	Value    string
	Language string
	Datatype Term
}

func NewLiteral(value string) (term Term) {
	return Term(&Literal{Value: value})
}

func NewLiteralWithLanguage(value string, language string) (term Term) {
	return Term(&Literal{Value: value, Language: language})
}

func NewLiteralWithDatatype(value string, datatype Term) (term Term) {
	return Term(&Literal{Value: value, Datatype: datatype})
}

func NewLiteralWithLanguageAndDatatype(value string, language string, datatype Term) (term Term) {
	return Term(&Literal{Value: value, Language: language, Datatype: datatype})
}

func (term Literal) String() (str string) {
	str = fmt.Sprintf("\"%s\"", term.Value)

	if term.Language != "" {
		str += "@" + term.Language
	} else if term.Datatype != nil {
		str += "^^" + term.Datatype.String()
	}

	return str
}

func (term Literal) Equal(other Term) bool {
	spec, ok := other.(*Literal)
	if !ok {
		return false
	}

	if term.Value != spec.Value {
		return false
	}

	if term.Language != spec.Language {
		return false
	}

	if (term.Datatype == nil && spec.Datatype != nil) || (term.Datatype != nil && spec.Datatype == nil) {
		return false
	}

	if term.Datatype != nil && spec.Datatype != nil && !term.Datatype.Equal(spec.Datatype) {
		return false
	}

	return true
}

type BlankNode struct {
	ID string
}

func NewBlankNode(id string) (term Term) {
	return Term(&BlankNode{ID: id})
}

func NewAnonNode() (term Term) {
	return NewBlankNode(fmt.Sprintf("b%d", <-newNodeIDs))
}

func (term BlankNode) String() (str string) {
	return "_:" + term.ID
}

func (term BlankNode) Equal(other Term) bool {
	if spec, ok := other.(*BlankNode); ok {
		return term.ID == spec.ID
	}

	return false
}
