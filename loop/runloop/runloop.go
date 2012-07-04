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
