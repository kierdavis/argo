/*
	Copyright (c) 2012 Kier Davis

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
	associated documentation files (the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial
	portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
	NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
	NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
	OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

// Package argo is an RDF manipulation, parsing and serialisation library. See
// https://github.com/kierdavis/argo for documentation and usage.
package argo

import (
	"io"
	"path"
	"strings"
)

// A Parser is a function that can parse a particular representation of RDF and stream the triples
// on a channel.
type Parser func(io.Reader, chan *Triple, chan error, map[string]string)

// A Serializer is a function that recieves triples sent along a channel and serializes them into a
// particular representation of RDF.
type Serializer func(io.Writer, chan *Triple, chan error, map[string]string)

// A Store is a container for RDF triples. For example, it could be backed by a flat list or a
// relational database.
type Store interface {
	// Method Add should add the given triple to the store.
	Add(*Triple)

	// Method Remove should remove the given triple from the store.
	Remove(*Triple)

	// Method Clear should remove all triples from the store.
	Clear()

	// Method Num should return the number of triples in the store.
	Num() int

	// Method IterTriples should return a channel that will yield the triples of the store. The
	// channel should be closed by this method when iteration is completed.
	IterTriples() chan *Triple

	// Method Filter should return a channel that will yield all matching triples of the graph. A
	// nil value passed means that the check for this term is skipped; else the triples returned
	// must have the same terms as the corresponding arguments.
	Filter(Term, Term, Term) chan *Triple
}

// A Format represents an RDF format.
type Format struct {
	// An identifier for this format (e.g. 'ntriples').
	ID string

	// A human-readable name for this format (e.g. 'N-Triples').
	Name string

	// The preferred MIME type for this format.
	PreferredMIMEType string

	// The preferred file extension for this format.
	PreferredExtension string

	// Other MIME types that will be mapped to this format.
	OtherMIMETypes []string

	// Other file extensions that will be mapped to this format.
	OtherExtensions []string

	// The Parser for this format.
	Parser Parser

	// The Serializer for this format.
	Serializer Serializer
}

// A map from format IDs to the corresponding Format objects.
var Formats = map[string]*Format{
	// http://www.w3.org/TR/REC-rdf-syntax/
	"rdfxml": &Format{
		ID:                 "rdfxml",
		Name:               "RDF/XML",
		PreferredMIMEType:  "application/rdf+xml",
		PreferredExtension: ".rdf",
		OtherMIMETypes:     []string{"application/xml", "text/xml"},
		OtherExtensions:    []string{".xml"},
		Parser:             ParseRDFXML,
		Serializer:         SerializeRDFXML,
	},

	// http://www.w3.org/2001/sw/RDFCore/ntriples/
	"ntriples": &Format{
		ID:                 "ntriples",
		Name:               "N-triples",
		PreferredMIMEType:  "text/plain",
		PreferredExtension: ".nt",
		OtherMIMETypes:     []string{"text/ntriples", "text/x-ntriples"},
		OtherExtensions:    []string{".txt"},
		Parser:             ParseNTriples,
		Serializer:         SerializeNTriples,
	},

	// http://docs.api.talis.com/platform-api/output-types/rdf-json
	"json": &Format{
		ID:                 "json",
		Name:               "RDF/JSON",
		PreferredMIMEType:  "application/rdf+json",
		PreferredExtension: ".json",
		OtherMIMETypes:     []string{"application/json", "text/json"},
		OtherExtensions:    []string{},
		Parser:             nil,
		Serializer:         SerializeJSON,
	},

	// http://www.w3.org/TeamSubmission/turtle/
	"turtle": &Format{
		ID:                 "turtle",
		Name:               "Turtle",
		PreferredMIMEType:  "text/turtle",
		PreferredExtension: ".ttl",
		OtherMIMETypes:     []string{},
		OtherExtensions:    []string{},
		Parser:             nil,
		Serializer:         SerializeTurtle,
	},

	// No specs yet
	"rdfz": &Format{
		ID:                 "rdfz",
		Name:               "RDFZ",
		PreferredMIMEType:  "application/x-rdf-compressed",
		PreferredExtension: ".rdfz",
		OtherMIMETypes:     []string{},
		OtherExtensions:    []string{},
		Parser:             nil,
		Serializer:         SerializeRDFZ,
	},

	// No specs yet
	"squirtle": &Format{
		ID:                 "squirtle",
		Name:               "Squirtle",
		PreferredMIMEType:  "text/x-squirtle",
		PreferredExtension: ".squirtle",
		OtherMIMETypes:     []string{},
		OtherExtensions:    []string{".sqtl"},
		Parser:             ParseSquirtle,
		Serializer:         SerializeSquirtle,
	},
}

// Function Parsers returns a map of format IDs to Formats that have an associated parser.
func Parsers() (parsers map[string]*Format) {
	parsers = make(map[string]*Format, 0)

	for id, format := range Formats {
		if format.Parser != nil {
			parsers[id] = format
		}
	}

	return parsers
}

// Function Serializers returns a map of format IDs to Formats that have an associated serializer.
func Serializers() (serializers map[string]*Format) {
	serializers = make(map[string]*Format, 0)

	for id, format := range Formats {
		if format.Parser != nil {
			serializers[id] = format
		}
	}

	return serializers
}

// Function FormatFromMIMEType takes a MIME type and returns the Format it represents, or nil if it
// could not be determined.
func FormatFromMIMEType(mimeType string) (format *Format) {
	for _, format = range Formats {
		if format.PreferredMIMEType == mimeType {
			return format
		}

		for _, m := range format.OtherMIMETypes {
			if m == mimeType {
				return format
			}
		}
	}

	return nil
}

// Function FormatFromFilename takes a filename and returns a Format based on its extension, or nil
// if it could not be determined.
func FormatFromFilename(filename string) (format *Format) {
	ext := path.Ext(filename)

	for _, format = range Formats {
		if format.PreferredExtension == ext {
			return format
		}

		for _, e := range format.OtherExtensions {
			if e == ext {
				return format
			}
		}
	}

	return nil
}

// Function SplitPrefix takes a given URI and splits it into a base URI and a local name (suitable
// for using as a qname in XML).
func SplitPrefix(uri string) (base string, name string) {
	index := strings.LastIndex(uri, "#") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	index = strings.LastIndex(uri, "/") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	return "", uri
}
