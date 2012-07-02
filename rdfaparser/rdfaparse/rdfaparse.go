package main

import (
	"fmt"
	"github.com/kierdavis/argo"
	"github.com/kierdavis/argo/rdfaparser"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Not enough arguments\nusage: %s <url>\n", os.Args[0])
		os.Exit(2)
	}

	graph := argo.NewGraph(argo.NewListStore())
	p := rdfaparser.NewRDFAParser(os.Args[1])
	err := graph.ParseHTTP(p, os.Args[1], "text/html")

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	err = graph.Serialize(argo.SerializeNTriples, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
