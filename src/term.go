package rdflib

import (
	"fmt"
	"rand"
)

const (
	TERM_NULL = iota
	TERM_RESOURCE = iota
	TERM_NODE = iota
	TERM_LITERAL = iota
)

type Term struct {
	Type int
	Value string
	Language string
	Datatype *Term
	prefix string // used internally
	name string
}

func CreateResource(uri string) (term *Term) {
	return &Term{
		Type: TERM_RESOURCE,
		Value: uri,
	}
}

func CreateNode(id string) (term *Term) {
	return &Term{
		Type: TERM_NODE,
		Value: id,
	}
}

func CreateBlankNode() (term *Term) {
	id := fmt.Sprintf("bn%d", rand.Int31())
	return CreateNode(id)
}

func CreateLiteral(text string) (term *Term) {
	return &Term{
		Type: TERM_LITERAL,
		Value: text,
		Language: "",
		Datatype: nil,
	}
}

func CreateLiteralWithLanguage(text string, language string) (term *Term) {
	term = CreateLiteral(text)
	term.Language = language
	return term
}

func CreateLiteralWithDatatype(text string, datatype *Term) (term *Term) {
	term = CreateLiteral(text)
	term.Datatype = datatype
	return term
}

func (term *Term) String() (str string) {
	switch term.Type {
		case TERM_RESOURCE:
			return fmt.Sprintf("<%s>", term.Value)
		
		case TERM_NODE:
			return fmt.Sprintf("_:%s", term.Value)
		
		case TERM_LITERAL:
			if term.Language != "" {
				return fmt.Sprintf("\"%s\"@%s", term.Value, term.Language)
			} else if term.Datatype != nil {
				return fmt.Sprintf("\"%s\"^^%s", term.Value, term.Datatype.String())
			} else {
				return fmt.Sprintf("\"%s\"", term.Value)
			}
	}

	return ""
}

func (term *Term) EqualTo(other *Term) (isEqual bool) {
	return term.Type == other.Type && term.Value == other.Value && term.Language == other.Language && term.Datatype == other.Datatype
}