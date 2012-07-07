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
	"github.com/kierdavis/ansi"
	"github.com/kierdavis/argo"
	"github.com/kierdavis/argo/squirtle"
	"github.com/kierdavis/argparse"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var TriplesProcessed uint
var BNodesRewritten uint

var FormatNames = []string{
	"ntriples",
	"rdfxml",
	"squirtle",
}

var StdoutLock sync.Mutex

type Args struct {
	OutFile             string
	URLs                []string
	Files               []string
	OutputFormat        string
	InputFormat         string
	StdinFormat         string
	RewriteBNodesPrefix string
}

func msg(style ansi.Attribute, format string, args ...interface{}) {
	StdoutLock.Lock()
	defer StdoutLock.Unlock()

	ansi.Fprintf(os.Stderr, style, format, args...)
}

func pipe(src chan *argo.Triple, dest chan *argo.Triple) {
	for triple := range src {
		dest <- triple
	}
}

func determineParserByFormat(format string) (parser argo.Parser, mimetype string) {
	switch format {
	case "ntriples":
		return argo.ParseNTriples, "text/plain"

	case "squirtle":
		return squirtle.ParseSquirtle, "text/x-squirtle"
	}

	return argo.ParseRDFXML, "application/rdf+xml"
}

func determineParserByExtension(path string) (parser argo.Parser, mimetype string) {
	if strings.HasSuffix(path, ".nt") || strings.HasSuffix(path, ".txt") {
		return argo.ParseNTriples, "text/plain"
	}

	if strings.HasSuffix(path, ".squirtle") {
		return squirtle.ParseSquirtle, "text/x-squirtle"
	}

	return argo.ParseRDFXML, "application/rdf+xml"
}

func determineSerializerByFormat(format string) (serializer argo.Serializer) {
	switch format {
	case "ntriples":
		return argo.SerializeNTriples

	case "turtle":
		return argo.SerializeTurtle

	case "json":
		return argo.SerializeJSON

	case "squirtle":
		return squirtle.SerializeSquirtle
	}

	return argo.SerializeRDFXML
}

func determineSerializerByExtension(path string) (serializer argo.Serializer) {
	if strings.HasSuffix(path, ".nt") || strings.HasSuffix(path, ".txt") {
		return argo.SerializeNTriples
	}

	if strings.HasSuffix(path, ".ttl") {
		return argo.SerializeTurtle
	}

	if strings.HasSuffix(path, ".json") {
		return argo.SerializeJSON
	}

	if strings.HasSuffix(path, ".squirtle") {
		return squirtle.SerializeSquirtle
	}

	return argo.SerializeRDFXML
}

func read(output chan *argo.Triple, errorOutput chan error, prefixMap map[string]string, args *Args) {
	// Concurrent loading, gives a minimal speed gain:

	var wg sync.WaitGroup

	for _, url := range args.URLs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			var parser argo.Parser
			var mimetype string

			if args.InputFormat != "" {
				parser, mimetype = determineParserByFormat(args.InputFormat)
			} else {
				parser, mimetype = determineParserByExtension(url)
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				errorOutput <- fmt.Errorf("Error when preparing to fetch '%s': %s", url, err.Error())
				return
			}

			req.Header.Add("Accept", mimetype)

			msg(ansi.Blue, "Fetching '%s'...\n", url)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errorOutput <- fmt.Errorf("Error when fetching '%s': %s", url, err.Error())
				return
			}
			defer resp.Body.Close()

			msg(ansi.Blue, "Parsing '%s'...\n", url)
			tripleChan := make(chan *argo.Triple)
			errChan := make(chan error)
			go parser(resp.Body, tripleChan, errChan, prefixMap)

			wg.Add(1)
			go func() {
				pipe(tripleChan, output)
				wg.Done()
			}()

			err = <-errChan
			if err != nil {
				errorOutput <- fmt.Errorf("Error when parsing '%s': %s", url, err.Error())
				return
			}

			msg(ansi.Blue, "Parsed '%s' successfully!\n", url)
		}()
	}

	for _, file := range args.Files {
		wg.Add(1)

		if file == "-" {
			go func() {
				defer wg.Done()

				var parser argo.Parser

				if args.StdinFormat != "" {
					parser, _ = determineParserByFormat(args.StdinFormat)
				} else if args.InputFormat != "" {
					parser, _ = determineParserByFormat(args.InputFormat)
				} else {
					parser = argo.ParseRDFXML
				}

				msg(ansi.Blue, "Parsing standard input...\n")
				tripleChan := make(chan *argo.Triple)
				errChan := make(chan error)
				go parser(os.Stdin, tripleChan, errChan, prefixMap)

				wg.Add(1)
				go func() {
					pipe(tripleChan, output)
					wg.Done()
				}()

				err := <-errChan
				if err != nil {
					errorOutput <- fmt.Errorf("Error when parsing standard input: %s", err.Error())
					return
				}

				msg(ansi.Blue, "Parsed standard input successfully!\n")
			}()

		} else {
			matches, err := filepath.Glob(file)
			if err != nil {
				errorOutput <- fmt.Errorf("Error when globbing '%s': %s", file, err.Error())
				continue
			}

			for _, match := range matches {
				go func() {
					defer wg.Done()

					var parser argo.Parser

					if args.InputFormat != "" {
						parser, _ = determineParserByFormat(args.InputFormat)
					} else {
						parser, _ = determineParserByExtension(match)
					}

					f, err := os.Open(match)
					if err != nil {
						errorOutput <- fmt.Errorf("Error when opening '%s' for reading: %s", match, err.Error())
						return
					}
					defer f.Close()

					msg(ansi.Blue, "Parsing '%s'...\n", match)
					tripleChan := make(chan *argo.Triple)
					errChan := make(chan error)

					wg.Add(1)
					go func() {
						pipe(tripleChan, output)
						wg.Done()
					}()

					go parser(f, tripleChan, errChan, prefixMap)

					err = <-errChan
					if err != nil {
						errorOutput <- fmt.Errorf("Error when parsing '%s': %s", match, err.Error())
						return
					}

					msg(ansi.Blue, "Parsed '%s' successfully!\n", match)
				}()
			}
		}
	}

	wg.Wait()
	close(output)
	close(errorOutput)
}

func rewriteBNode(termRef *argo.Term, prefix string) {
	if bnode, ok := (*termRef).(*argo.BlankNode); ok {
		*termRef = argo.NewResource(prefix + bnode.ID)
	}

	BNodesRewritten++
}

func main() {
	startTime := time.Now()

	args := &Args{
		OutFile: "-",
	}

	p := argparse.New("A tool for manipulating RDF files.")
	p.Option('o', "output", "OutFile", 1, argparse.Store, "FILENAME", "The file to write output to. Default: standard output.")
	p.Option('u', "url", "URLs", 1, argparse.Append, "URL", "A URL to download from and add to the graph. Can be used multiple times. Default: no URLs will be downloaded.")
	p.Option('O', "output-format", "OutputFormat", 1, argparse.Choice(argparse.Store, FormatNames...), "FORMAT", "The format to write output to. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('I', "input-format", "InputFormat", 1, argparse.Choice(argparse.Store, FormatNames...), "FORMAT", "The format to parse all input sources as. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('i', "stdin-format", "StdinFormat", 1, argparse.Choice(argparse.Store, FormatNames...), "FORMAT", "The format to parse stdin as. The formats for all other sources (files and URLs) are still determined by their file extensions. Default: rdfxml.")
	p.Option(0, "rewrite-bnodes", "RewriteBNodesPrefix", 1, argparse.Store, "URIPREFIX", "Replace all blank nodes with a URI reference consisting of the given prefix and the blank node's identifier. Example (--rewrite-bnodes http://example.org/bnodes/) _:foobar -> http://example.org/bnodes/foobar. Default: no rewriting.")
	p.Argument("Files", argparse.ZeroOrMore, argparse.Store, "filename", "Files to parse and add to the graph.")
	err := p.Parse(args)

	if err != nil {
		ansi.Fprintf(os.Stderr, ansi.RedBold, "Error when parsing arguments: %s\n", err.Error())
		os.Exit(1)
	}

	// =============================================================================================

	tripleChan := make(chan *argo.Triple)
	errChan := make(chan error)
	graph := argo.NewGraph(argo.NewListStore())

	go read(tripleChan, errChan, graph.Prefixes, args)
	//go graph.LoadFromChannel(tripleChan)

	go func() {
		if args.RewriteBNodesPrefix == "" {
			for triple := range tripleChan {
				graph.Add(triple)
				TriplesProcessed++
			}

		} else {
			for triple := range tripleChan {
				rewriteBNode(&triple.Subject, args.RewriteBNodesPrefix)
				rewriteBNode(&triple.Object, args.RewriteBNodesPrefix)

				graph.Add(triple)
				TriplesProcessed++
			}
		}
	}()

	wasErrors := false
	for err = range errChan {
		wasErrors = true
		msg(ansi.RedBold, "%s\n", err.Error())
	}

	if wasErrors && graph.Num() == 0 {
		// Only exit if _all_ parses failed
		os.Exit(1)
	}

	// =============================================================================================

	var output io.Writer
	var serializer argo.Serializer

	if args.OutFile == "-" {
		output = os.Stdout
		serializer = argo.SerializeRDFXML

	} else {
		output, err = os.Create(args.OutFile)
		if err != nil {
			msg(ansi.RedBold, "Error when opening '%s' for writing: %s\n", args.OutFile, err.Error())
			os.Exit(1)
		}

		serializer = determineSerializerByExtension(args.OutFile)
	}

	if args.OutputFormat != "" {
		serializer = determineSerializerByFormat(args.OutputFormat)
	}

	msg(ansi.Blue, "Serializing...\n")
	err = graph.Serialize(serializer, output)

	if err != nil {
		ansi.Fprintf(os.Stderr, ansi.RedBold, "Error when serializing: %s\n", args.OutFile, err.Error())
		os.Exit(1)
	}

	msg(ansi.Blue, "Serialized!\n")

	ms := float64(time.Since(startTime).Nanoseconds()) / 1000000.0
	msg(ansi.Blue, "\n%d triples processed in %.3f seconds (%.3f ms)\n", TriplesProcessed, ms/1000.0, ms)

	if args.RewriteBNodesPrefix != "" {
		msg(ansi.Blue, "%d blank nodes rewritten\n", BNodesRewritten)
	}
}
