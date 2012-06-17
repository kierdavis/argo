package argo

import (
	"bytes"
	"strings"
	"testing"
)

var testCases = map[string]*Triple{
	"<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": NewTriple(
		NewResource("http://example.org/resource1"),
		NewResource("http://example.org/property"),
		NewResource("http://example.org/resource2")),

	"_:anon <http://example.org/property> <http://example.org/resource2> .": NewTriple(NewNode("anon"),
		NewResource("http://example.org/property"),
		NewResource("http://example.org/resource2")),

	"<http://example.org/resource1> <http://example.org/property> _:anon .": NewTriple(NewResource("http://example.org/resource1"),
		NewResource("http://example.org/property"),
		NewNode("anon")),

	" 	 <http://example.org/resource3> 	 <http://example.org/property>	 <http://example.org/resource2> 	.": NewTriple(NewResource("http://example.org/resource3"),
		NewResource("http://example.org/property"),
		NewResource("http://example.org/resource2")),

	"<http://example.org/resource7> <http://example.org/property> \"simple literal\" .": NewTriple(NewResource("http://example.org/resource7"),
		NewResource("http://example.org/property"),
		NewLiteral("simple literal")),

	`<http://example.org/resource8> <http://example.org/property> "backslash:\\" .`: NewTriple(NewResource("http://example.org/resource8"),
		NewResource("http://example.org/property"),
		NewLiteral("backslash:\\")),

	`<http://example.org/resource9> <http://example.org/property> "dquote:\"" .`: NewTriple(NewResource("http://example.org/resource9"),
		NewResource("http://example.org/property"),
		NewLiteral("dquote:\"")),

	`<http://example.org/resource10> <http://example.org/property> "newline:\n" .`: NewTriple(NewResource("http://example.org/resource10"),
		NewResource("http://example.org/property"),
		NewLiteral("newline:\n")),

	`<http://example.org/resource11> <http://example.org/property> "return\r" .`: NewTriple(NewResource("http://example.org/resource11"),
		NewResource("http://example.org/property"),
		NewLiteral("return\r")),

	`<http://example.org/resource12> <http://example.org/property> "tab:\t" .`: NewTriple(NewResource("http://example.org/resource12"),
		NewResource("http://example.org/property"),
		NewLiteral("tab:\t")),

	`<http://example.org/resource16> <http://example.org/property> "\u00E9" .`: NewTriple(NewResource("http://example.org/resource16"),
		NewResource("http://example.org/property"),
		NewLiteral("\u00E9")),

	`<http://example.org/resource30> <http://example.org/property> "chat"@fr .`: NewTriple(NewResource("http://example.org/resource30"),
		NewResource("http://example.org/property"),
		NewLiteralWithLanguage("chat","fr")),

	`<http://example.org/resource31> <http://example.org/property> "chat"@en .`: NewTriple(NewResource("http://example.org/resource31"),
		NewResource("http://example.org/property"),
		NewLiteralWithLanguage("chat","en")),

	"# this is a comment \n<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": NewTriple(NewResource("http://example.org/resource1"),
		NewResource("http://example.org/property"),
		NewResource("http://example.org/resource2")),

	"# this is a comment \n   # another comment \n<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": NewTriple(NewResource("http://example.org/resource1"),
		NewResource("http://example.org/property"),
		NewResource("http://example.org/resource2")),

	"<http://example.org/resource7> <http://example.org/property> \"typed literal\"^^<http://example.org/datatype1> .": NewTriple(NewResource("http://example.org/resource7"),
		NewResource("http://example.org/property"),
		NewLiteralWithDatatype("typed literal", NewResource("http://example.org/datatype1"))),
}

var negativeCases = map[string]error{
	"<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ":   ErrUnterminatedTriple,
	"<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ,":  ErrUnexpectedCharacter,
	"<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ..": ErrUnexpectedCharacter,
	"http://example.org/resource1> <http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1 <http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1><http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1> <http://example.org/property><http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1> http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1> <http://example.org/property <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1> <http://example.org/property> http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2.":    ErrUnexpectedEOF,
	"<http://example.org/resource1> \n<http://example.org/property> <http://example.org/resource2>.": ErrUnexpectedCharacter,
	"_:foo\n <http://example.org/property> <http://example.org/resource2>.":                          ErrUnexpectedCharacter,
	"_:0abc <http://example.org/property> <http://example.org/resource2>.":                           ErrUnexpectedCharacter,
	"_abc <http://example.org/property> <http://example.org/resource2>.":                             ErrUnexpectedCharacter,
	"_:a-bc <http://example.org/property> <http://example.org/resource2>.":                           ErrUnexpectedCharacter,
	"_:abc<http://example.org/property> <http://example.org/resource2>.":                             ErrUnexpectedCharacter,
	"_:abc <http://example.org/property> \"foo\"@ .":                                                 ErrUnexpectedCharacter,
	"_:abc <http://example.org/property> \"foo\"^ .":                                                 ErrUnexpectedCharacter,
	"_:abc <http://example.org/property> \"foo\"^^< .":                                               ErrUnexpectedCharacter,
	"_:abc <http://example.org/property> \"foo\"^^<> .":                                              ErrUnexpectedCharacter,
	"_:abc <> _:abc .":                                                                               ErrUnexpectedCharacter,
	"_:abc < > _:abc .":                                                                              ErrUnexpectedCharacter,
}

func TestRead(t *testing.T) {
	for ntriple, expected := range testCases {
		r := NewReader(strings.NewReader(ntriple))
		triple, err := r.Read()
		if err != nil {
			t.Errorf("Expected %s but got error %s", *expected, err)
		}

		if triple == nil {
			t.Errorf("Expected %s but got nil triple", *expected)
		} 


		if !triple.Equal(expected) {
			t.Errorf("Expected %#v but got %#v", expected, triple)
		}
	}
}

func TestReadMultiple(t *testing.T) {
	var ntriples bytes.Buffer
	var triples []*Triple

	for ntriple, triple := range testCases {
		ntriples.WriteString(ntriple)
		ntriples.WriteRune('\n')
		triples = append(triples, triple)
	}

	count := 0
	r := NewReader(strings.NewReader(ntriples.String()))
	triple, err := r.Read()
	for err == nil {
		if !triple.Equal(triples[count]) {
			t.Errorf("Expected %s but got %s", triples[count], triple)
			break
		}

		count++
		triple, err = r.Read()
	}

	if count != len(triples) {
		t.Errorf("Expected %d but only parsed %d triples", len(triples), count)

	}

}

func TestReadErrors(t *testing.T) {

	for ntriple, expected := range negativeCases {
		r := NewReader(strings.NewReader(ntriple))
		_, err := r.Read()

		if err == nil {
			t.Errorf("Expected %s for %s but no error reported", expected, ntriple)
		} else if err.(*ParseError).Err != expected {
			t.Errorf("Expected %s for %s but got error %s", expected, ntriple, err.(*ParseError).Err)
		}
	}
}
