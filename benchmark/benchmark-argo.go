package main

import (
	"argo"
	"os"
)

func main() {
	graph := argo.CreateGraphWithMemoryStore()
	graph.Bind(argo.FOAF.Base(), "foaf")

	subj := argo.CreateResource("http://kierdavis.tklapp.com/me")
	EXAMPLE := graph.Bind("http://example.org/foo/", "ex")

	graph.AddTriple(subj, argo.FOAF.Get("name"), argo.CreateLiteralWithDatatype("Kier", EXAMPLE.Get("name")))
	graph.AddTriple(subj, argo.FOAF.Get("made"), argo.CreateResource("http://kierdavis.tklapp.com/"))

	graph.WriteXML(os.Stdout)
}