package main

import (
	"fmt"
	"github.com/kierdavis/argo"
	"io/ioutil"
	"os"
)

func main() {
	input, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	go yyParse(&yyLex{newLexer("LoopC", string(input))})

	graph := argo.NewGraph(argo.NewListStore())

	for f := range parserOutput {
		f.ToRDF(graph)
	}

	graph.Serialize(argo.SerializeNTriples, os.Stdout)
}
