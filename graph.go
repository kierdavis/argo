package argo

import (
	"io"
	"strings"
)

type Parser func(io.Reader, chan *Triple, chan error)
type Serializer func(io.Writer, chan *Triple, chan error)

type Store interface {
	Add(*Triple) int
	Remove(*Triple)
	RemoveIndex(int)
	Clear()
	Num() int
	IterTriples() chan *Triple
}

type Graph struct {
	store    Store
	prefixes map[string]string
}

func NewGraph(store Store) (graph *Graph) {
	return &Graph{
		store:    store,
		prefixes: map[string]string{"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf"},
	}
}

func (graph *Graph) Bind(uri string, prefix string) (ns Namespace) {
	graph.prefixes[uri] = prefix
	return NewNamespace(uri)
}

func (graph *Graph) LookupAndBind(prefix string) (ns Namespace, err error) {
	uri, err := LookupPrefix(prefix)
	if err != nil {
		return ns, err
	}

	return graph.Bind(uri, prefix), nil
}

func (graph *Graph) Add(triple *Triple) (index int) {
	return graph.store.Add(triple)
}

func (graph *Graph) AddTriple(subject Term, predicate Term, object Term) (index int) {
	return graph.store.Add(NewTriple(subject, predicate, object))
}

func (graph *Graph) AddQuad(subject Term, predicate Term, object Term, context Term) (index int) {
	return graph.store.Add(NewQuad(subject, predicate, object, context))
}

func (graph *Graph) Remove(triple *Triple) {
	graph.store.Remove(triple)
}

func (graph *Graph) RemoveIndex(index int) {
	graph.store.RemoveIndex(index)
}

func (graph *Graph) RemoveTriple(subject Term, predicate Term, object Term) {
	graph.store.Remove(NewTriple(subject, predicate, object))
}

func (graph *Graph) RemoveQuad(subject Term, predicate Term, object Term, context Term) {
	graph.store.Remove(NewQuad(subject, predicate, object, context))
}

func (graph *Graph) Clear() {
	graph.store.Clear()
}

func (graph *Graph) Num() (n int) {
	return graph.store.Num()
}

func (graph *Graph) IterTriples() (ch chan *Triple) {
	return graph.store.IterTriples()
}

func (graph *Graph) Parse(parser Parser, r io.Reader) (err error) {
	tripleChan := make(chan *Triple)
	errChan := make(chan error)

	go parser(r, tripleChan, errChan)

	for triple := range tripleChan {
		graph.Add(triple)
	}

	return <-errChan
}

func (graph *Graph) Serialize(serializer Serializer, w io.Writer) (err error) {
	errChan := make(chan error)

	serializer(w, graph.IterTriples(), errChan)

	return <-errChan
}

func splitPrefix(uri string) (base string, name string) {
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
