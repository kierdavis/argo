// Package argo is an RDF manipulation, parsing and serialisation library. See
// https://github.com/kierdavis/argo for documentation and usage.
package argo

import (
	"io"
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
