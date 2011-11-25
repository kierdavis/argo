package main

import (
	"fmt"
	"os"
	"rdflib"
)

func main() {
	subj := rdflib.CreateResource("http://kierdavis.tklapp.com/me")

	graph := rdflib.CreateGraphWithMemoryStore()
	FOAF := graph.BindNamespace("http://xmlns.com/foaf/0.1/", "foaf")
	EXAMPLE := graph.BindNamespace("http://example.org/foo/", "ex")

	graph.AddTriple(subj, FOAF.Get("name"), rdflib.CreateLiteralWithDatatype("Kier", EXAMPLE.Get("name")))
	graph.AddTriple(subj, FOAF.Get("made"), rdflib.CreateResource("http://kierdavis.tklapp.com/"))

	fmt.Println()
	graph.WriteNTriples(os.Stdout)

	fmt.Println()
	graph.WriteXML(os.Stdout)
}
