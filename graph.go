package argo

import (
	"io"
)

// A Graph wraps a Store and provides extra convenience methods.
type Graph struct {
	// The associated triple store.
	store Store

	// The prefix map.
	prefixes map[string]string
}

// Function NewGraph creates and returns a new graph.
func NewGraph(store Store) (graph *Graph) {
	return &Graph{
		store:    store,
		prefixes: map[string]string{"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf"},
	}
}

// Function Bind adds the given URI/prefix mapping to the internal map, and returns the uri wrapped
// in a Namespace for your convenience.
func (graph *Graph) Bind(uri string, prefix string) (ns Namespace) {
	graph.prefixes[uri] = prefix
	return NewNamespace(uri)
}

// Function LookupAndBind looks up the prefix using the prefix.cc service, then maps the prefix to
// the returned URI and returns the URI wrapped in a Namespace for your convenience.
func (graph *Graph) LookupAndBind(prefix string) (ns Namespace, err error) {
	uri, err := LookupPrefix(prefix)
	if err != nil {
		return ns, err
	}

	return graph.Bind(uri, prefix), nil
}

// Function Add adds the given triple to the graph and returns its index.
func (graph *Graph) Add(triple *Triple) (index int) {
	return graph.store.Add(triple)
}

// Function AddTriple creates a triple from the arguments and adds it to the graph.
func (graph *Graph) AddTriple(subject Term, predicate Term, object Term) (index int) {
	return graph.store.Add(NewTriple(subject, predicate, object))
}

// Function AddQuad creates a quad from the arguments and adds it to the graph.
func (graph *Graph) AddQuad(subject Term, predicate Term, object Term, context Term) (index int) {
	return graph.store.Add(NewQuad(subject, predicate, object, context))
}

// Function Remove removes the given triple from the graph, if it exists.
func (graph *Graph) Remove(triple *Triple) {
	graph.store.Remove(triple)
}

// Function RemoveIndex removes the triple with the given index from the graph, if it exists.
func (graph *Graph) RemoveIndex(index int) {
	graph.store.RemoveIndex(index)
}

// Function Remove removes the given triple from the graph, if it exists.
func (graph *Graph) RemoveTriple(subject Term, predicate Term, object Term) {
	graph.store.Remove(NewTriple(subject, predicate, object))
}

// Function Remove removes the given quad from the graph, if it exists.
func (graph *Graph) RemoveQuad(subject Term, predicate Term, object Term, context Term) {
	graph.store.Remove(NewQuad(subject, predicate, object, context))
}

// Function Clear clears the graph.
func (graph *Graph) Clear() {
	graph.store.Clear()
}

// Function Num returns the number of triples in the graph.
func (graph *Graph) Num() (n int) {
	return graph.store.Num()
}

// Function IterTriples returns a channel that will yield the triples of the graph. The channel will
// be closed when iteration is completed.
func (graph *Graph) IterTriples() (ch chan *Triple) {
	return graph.store.IterTriples()
}

// Function Parse uses the specified Parser to parse an RDF file from an io.Reader.
func (graph *Graph) Parse(parser Parser, r io.Reader) (err error) {
	tripleChan := make(chan *Triple)
	errChan := make(chan error)

	go parser(r, tripleChan, errChan)

	for triple := range tripleChan {
		graph.Add(triple)
	}

	return <-errChan
}

// Function Serialize uses the specified Serializer to serialize an RDF file to an io.Writer.
func (graph *Graph) Serialize(serializer Serializer, w io.Writer) (err error) {
	errChan := make(chan error)

	serializer(w, graph.IterTriples(), errChan)

	return <-errChan
}
