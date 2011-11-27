package main

import (
	"fmt"
	"os"
	"argo"
)

func main() {
	subj := argo.CreateResource("http://kierdavis.tklapp.com/me")

	graph := argo.CreateGraphWithMemoryStore()
	FOAF, err := graph.LookupAndBind("foaf")
	if err != nil {panic(err)}
	//FOAF := graph.Bind("http://xmlns.com/foaf/0.1/", "foaf")
	EXAMPLE := graph.Bind("http://example.org/foo/", "ex")

	graph.AddTriple(subj, FOAF.Get("name"), argo.CreateLiteralWithDatatype("Kier", EXAMPLE.Get("name")))
	graph.AddTriple(subj, FOAF.Get("made"), argo.CreateResource("http://kierdavis.tklapp.com/"))

	fmt.Println()
	graph.WriteNTriples(os.Stdout)

	fmt.Println()
	graph.WriteXML(os.Stdout)
}
