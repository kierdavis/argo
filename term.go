package argo

import (
	"fmt"
	"github.com/kierdavis/goutil"
)

var newNodeIDs = util.UniqueInts()

// A Term is the value of a subject, predicate or object i.e. a IRI reference, blank node or
// literal.
type Term interface {
	// Method String should return the NTriples representation of this term.
	String() string

	// Method Equal should return whether this term is equal to another.
	Equal(Term) bool
}

// A Resource is an URI / IRI reference.
type Resource struct {
	URI string
}

// Function NewResource returns a new resource object.
func NewResource(uri string) (term Term) {
	return Term(&Resource{URI: uri})
}

// Method String returns the NTriples representation of this resource.
func (term Resource) String() (str string) {
	return fmt.Sprintf("<%s>", term.URI)
}

// Method Equal returns whether this resource is equal to another.
func (term Resource) Equal(other Term) bool {
	if spec, ok := other.(*Resource); ok {
		return term.URI == spec.URI
	}

	return false
}

// A Literal is a textual value, with an associated language or datatype.
type Literal struct {
	Value    string
	Language string
	Datatype Term
}

// Function NewLiteral returns a new literal with the given value.
func NewLiteral(value string) (term Term) {
	return Term(&Literal{Value: value})
}

// Function NewLiteralWithLanguage returns a new literal with the given value and language.
func NewLiteralWithLanguage(value string, language string) (term Term) {
	return Term(&Literal{Value: value, Language: language})
}

// Function NewLiteralWithDatatype returns a new literal with the given value and datatype.
func NewLiteralWithDatatype(value string, datatype Term) (term Term) {
	return Term(&Literal{Value: value, Datatype: datatype})
}

// Function NewLiteralWithLanguageAndDatatype returns a new literal with the given value, language
// and datatype. Technically a literal cannot have both a language and a datatype, but this function
// is provided to allow creation of literal in a context where this check has already been made,
// such as in a parser.
func NewLiteralWithLanguageAndDatatype(value string, language string, datatype Term) (term Term) {
	return Term(&Literal{Value: value, Language: language, Datatype: datatype})
}

// Method String returns the NTriples representation of this literal.
func (term Literal) String() (str string) {
	str = fmt.Sprintf("\"%s\"", term.Value)

	if term.Language != "" {
		str += "@" + term.Language
	} else if term.Datatype != nil {
		str += "^^" + term.Datatype.String()
	}

	return str
}

// Method Equal returns whether this literal is equivalent to another.
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

// A BlankNode is an RDF blank node i.e. an unqualified URI/IRI.
type BlankNode struct {
	ID string
}

// Function NewBlankNode returns a new blank node with the given ID.
func NewBlankNode(id string) (term Term) {
	return Term(&BlankNode{ID: id})
}

// Function NewAnonNode returns a new blank node with an automatically generated ID of the form
// 'bN', where N is a positive integer. The IDs generated are guaranteed to be unique across a
// single run of the program; however they are not unique between separate runs.
func NewAnonNode() (term Term) {
	return NewBlankNode(fmt.Sprintf("b%d", <-newNodeIDs))
}

// Method String returns the NTriples representation of the blank node.
func (term BlankNode) String() (str string) {
	return "_:" + term.ID
}

// Method Equal returns whether this blank node is equivalent to another.
func (term BlankNode) Equal(other Term) bool {
	if spec, ok := other.(*BlankNode); ok {
		return term.ID == spec.ID
	}

	return false
}
