package main

import (
	"fmt"
	"github.com/kierdavis/argo"
	"github.com/kierdavis/argo/loop"
	"os"
	"strings"
)

func main() {
	loop.Debug = true

	graph := loop.NewInterpreter()

	if len(os.Args) >= 3 {
		url := os.Args[2]
		if strings.HasPrefix(url, "http:") {
			graph.ParseHTTP(argo.ParseRDFXML, url, "application/rdf+xml")

		} else {
			graph.ParseFile(argo.ParseRDFXML, url)
		}
	}

	graph.Serialize(argo.SerializeNTriples, os.Stdout)

	funcTerm := argo.NewResource(os.Args[1])
	value, err := graph.Evaluate(funcTerm, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}

	fmt.Printf("%v\n", value)
}
