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
	"github.com/kierdavis/argparse"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	// These packages register their parsers/serializers in argo.Formats, so we don't actually need
	// to directly reference the package.
	_ "github.com/kierdavis/argo/rdfaparser"
)

var LookupCacheFile = filepath.Join(os.Getenv("HOME"), ".prefixes.gob")

var TriplesProcessed uint
var Rewritten uint

var Parsers, Serializers []string

func init() {
	for id := range argo.Parsers() {
		Parsers = append(Parsers, id)
	}

	for id := range argo.Serializers() {
		Serializers = append(Serializers, id)
	}

	sort.Strings(Parsers)
	sort.Strings(Serializers)
}

type Args struct {
	OutFile           string
	URLs              []string
	Files             []string
	OutputFormat      string
	InputFormat       string
	StdinFormat       string
	ShowFormats       bool
	Rewrites          []string
	SubjectRewrites   []string
	PredicateRewrites []string
	ObjectRewrites    []string
}

func init() {
	ansi.UseMutex = true
}

func msg(style ansi.Attribute, format string, args ...interface{}) {
	ansi.Fprintf(os.Stderr, style, format, args...)
}

func pipe(src chan *argo.Triple, dest chan *argo.Triple) {
	for triple := range src {
		dest <- triple
	}
}

func read(output chan *argo.Triple, errorOutput chan error, prefixMap map[string]string, args *Args) {
	// Concurrent loading, gives a minimal speed gain:

	var wg sync.WaitGroup

	for _, url := range args.URLs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			var format *argo.Format

			if args.InputFormat != "" {
				format = argo.Formats[args.InputFormat]
			} else {
				format = argo.FormatFromFilename(url)
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				errorOutput <- fmt.Errorf("Error when preparing to fetch '%s': %s", url, err.Error())
				return
			}

			req.Header.Add("Accept", format.PreferredMIMEType)

			msg(ansi.White, "Fetching '%s'...\n", url)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errorOutput <- fmt.Errorf("Error when fetching '%s': %s", url, err.Error())
				return
			}
			defer resp.Body.Close()

			msg(ansi.White, "Parsing '%s' as %s...\n", url, format.Name)
			tripleChan := make(chan *argo.Triple)
			errChan := make(chan error)

			wg.Add(1)
			go func() {
				pipe(tripleChan, output)
				wg.Done()
			}()

			go format.Parser(resp.Body, tripleChan, errChan, prefixMap)

			err = <-errChan
			if err != nil {
				errorOutput <- fmt.Errorf("Error when parsing '%s': %s", url, err.Error())
				return
			}

			msg(ansi.White, "Parsed '%s' successfully!\n", url)
		}()
	}

	for _, file := range args.Files {
		wg.Add(1)

		if file == "-" {
			go func() {
				defer wg.Done()

				var format *argo.Format

				if args.StdinFormat != "" {
					format = argo.Formats[args.StdinFormat]
				} else if args.InputFormat != "" {
					format = argo.Formats[args.InputFormat]
				} else {
					format = argo.Formats["rdfxml"]
				}

				msg(ansi.White, "Parsing standard input as %s...\n", format.Name)
				tripleChan := make(chan *argo.Triple)
				errChan := make(chan error)

				wg.Add(1)
				go func() {
					pipe(tripleChan, output)
					wg.Done()
				}()

				go format.Parser(os.Stdin, tripleChan, errChan, prefixMap)

				err := <-errChan
				if err != nil {
					errorOutput <- fmt.Errorf("Error when parsing standard input: %s", err.Error())
					return
				}

				msg(ansi.White, "Parsed standard input successfully!\n")
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

					var format *argo.Format

					if args.InputFormat != "" {
						format = argo.Formats[args.InputFormat]
					} else {
						format = argo.FormatFromFilename(match)
					}

					f, err := os.Open(match)
					if err != nil {
						errorOutput <- fmt.Errorf("Error when opening '%s' for reading: %s", match, err.Error())
						return
					}
					defer f.Close()

					msg(ansi.White, "Parsing '%s' as %s...\n", match, format.Name)
					tripleChan := make(chan *argo.Triple)
					errChan := make(chan error)

					wg.Add(1)
					go func() {
						pipe(tripleChan, output)
						wg.Done()
					}()

					go format.Parser(f, tripleChan, errChan, prefixMap)

					err = <-errChan
					if err != nil {
						errorOutput <- fmt.Errorf("Error when parsing '%s': %s", match, err.Error())
						return
					}

					msg(ansi.White, "Parsed '%s' successfully!\n", match)
				}()
			}
		}
	}

	wg.Wait()
	close(output)
	close(errorOutput)
}

type Rewrite struct {
	Regexp   *regexp.Regexp
	Template string
}

func (rewrite Rewrite) Apply(termPtr *argo.Term) {
	term := *termPtr
	termStr := ""

	switch realTerm := term.(type) {
	case *argo.Resource:
		termStr = realTerm.URI
	case *argo.BlankNode:
		termStr = "_:" + realTerm.ID
	case *argo.Literal:
		return
	}

	match := rewrite.Regexp.FindStringSubmatchIndex(termStr)
	if match != nil {
		resBytes := rewrite.Regexp.ExpandString(nil, rewrite.Template, termStr, match)
		resStr := string(resBytes)

		if len(resStr) >= 2 && resStr[0] == '_' && resStr[1] == ':' {
			*termPtr = argo.NewBlankNode(resStr[2:])
		} else {
			*termPtr = argo.NewResource(resStr)
		}

		Rewritten++
	}
}

func rewrite(termPtr *argo.Term, rewrites1, rewrites2 []Rewrite) {
	if rewrites1 != nil {
		for _, rewrite := range rewrites1 {
			rewrite.Apply(termPtr)
		}
	}

	if rewrites2 != nil {
		for _, rewrite := range rewrites2 {
			rewrite.Apply(termPtr)
		}
	}
}

func main() {
	argo.LoadLookupCache(LookupCacheFile)
	defer argo.SaveLookupCache(LookupCacheFile)

	startTime := time.Now()

	args := &Args{
		OutFile: "-",
	}

	p := argparse.New("A tool for manipulating RDF files.")
	p.Option('o', "output", "OutFile", 1, argparse.Store, "FILENAME", "The file to write output to. Default: standard output.")
	p.Option('u', "url", "URLs", 1, argparse.Append, "URL", "A URL to download from and add to the graph. Can be used multiple times. Default: no URLs will be downloaded.")
	p.Option('I', "input-format", "InputFormat", 1, argparse.Choice(argparse.Store, Parsers...), "FORMAT", "The format to parse all input sources as. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('i', "stdin-format", "StdinFormat", 1, argparse.Choice(argparse.Store, Parsers...), "FORMAT", "The format to parse stdin as. The formats for all other sources (files and URLs) are still determined by their file extensions. Default: rdfxml.")
	p.Option('O', "output-format", "OutputFormat", 1, argparse.Choice(argparse.Store, Serializers...), "FORMAT", "The format to write output to. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('F', "formats", "ShowFormats", 0, argparse.StoreConst(true), "", "Display a list of formats.")
	p.Option('r', "rewrite", "Rewrites", 2, argparse.Append, "FIND REPLACE", "Replaces all URIs and blank nodes that match the standard regular expression FIND with the URI REPLACE. Within REPLACE, patterns such as $1, $2 etc. expanding to the text of the first and second submatch respectively. This option can be used multiple times. Input and output strings that have the prefix '_:' are interpreted as blank nodes; otherwise they are URIs.")
	p.Option(0, "rewrite-subject", "SubjectRewrites", 2, argparse.Append, "FIND REPLACE", "Like -r/--rewrite, but only applies to subject terms.")
	p.Option(0, "rewrite-predicate", "PredicateRewrites", 2, argparse.Append, "FIND REPLACE", "Like -r/--rewrite, but only applies to predicate terms.")
	p.Option(0, "rewrite-object", "ObjectRewrites", 2, argparse.Append, "FIND REPLACE", "Like -r/--rewrite, but only applies to object terms.")
	p.Argument("Files", argparse.ZeroOrMore, argparse.Store, "filename", "Files to parse and add to the graph.")
	err := p.Parse(args)

	if err != nil {
		ansi.Fprintf(os.Stderr, ansi.RedBold, "Error when parsing arguments: %s\n", err.Error())
		os.Exit(1)
	}

	if args.ShowFormats {
		fmt.Printf("Input formats:\n")

		for _, id := range Parsers {
			fmt.Printf("  %s - %s\n", id, argo.Formats[id].Name)
		}

		fmt.Printf("\nOutput formats:\n")

		for _, id := range Serializers {
			fmt.Printf("  %s - %s\n", id, argo.Formats[id].Name)
		}

		return
	}

	// =============================================================================================

	var rewrites, subjectRewrites, predicateRewrites, objectRewrites []Rewrite

	if args.Rewrites != nil {
		rewrites = make([]Rewrite, len(args.Rewrites)/2)

		for i := 0; i < len(args.Rewrites); i += 2 {
			rewrites[i/2].Regexp = regexp.MustCompile(args.Rewrites[i])
			rewrites[i/2].Template = args.Rewrites[i+1]
		}
	}

	if args.SubjectRewrites != nil {
		subjectRewrites = make([]Rewrite, len(args.SubjectRewrites)/2)

		for i := 0; i < len(args.SubjectRewrites); i += 2 {
			subjectRewrites[i/2].Regexp = regexp.MustCompile(args.SubjectRewrites[i])
			subjectRewrites[i/2].Template = args.SubjectRewrites[i+1]
		}
	}

	if args.PredicateRewrites != nil {
		predicateRewrites = make([]Rewrite, len(args.PredicateRewrites)/2)

		for i := 0; i < len(args.PredicateRewrites); i += 2 {
			predicateRewrites[i/2].Regexp = regexp.MustCompile(args.PredicateRewrites[i])
			predicateRewrites[i/2].Template = args.PredicateRewrites[i+1]
		}
	}

	if args.ObjectRewrites != nil {
		objectRewrites = make([]Rewrite, len(args.ObjectRewrites)/2)

		for i := 0; i < len(args.ObjectRewrites); i += 2 {
			objectRewrites[i/2].Regexp = regexp.MustCompile(args.ObjectRewrites[i])
			objectRewrites[i/2].Template = args.ObjectRewrites[i+1]
		}
	}

	parseChan := make(chan *argo.Triple)
	serializeChan := make(chan *argo.Triple)
	errChan := make(chan error)
	prefixMap := make(map[string]string)

	var output io.Writer
	format := argo.Formats["rdfxml"]

	if args.OutFile == "-" {
		output = os.Stdout

	} else {
		output, err = os.Create(args.OutFile)
		if err != nil {
			msg(ansi.RedBold, "Error when opening '%s' for writing: %s\n", args.OutFile, err.Error())
			os.Exit(1)
		}

		format = argo.FormatFromFilename(args.OutFile)
	}

	if args.OutputFormat != "" {
		format = argo.Formats[args.OutputFormat]
	}

	msg(ansi.White, "Serializing as %s...\n", format.Name)
	go read(parseChan, errChan, prefixMap, args)
	go format.Serializer(output, serializeChan, errChan, prefixMap)

	go func() {
		for triple := range parseChan {
			rewrite(&triple.Subject, rewrites, subjectRewrites)
			rewrite(&triple.Predicate, rewrites, predicateRewrites)
			rewrite(&triple.Object, rewrites, objectRewrites)

			serializeChan <- triple
			TriplesProcessed++
		}
	}()

	for err = range errChan {
		msg(ansi.RedBold, "Error: %s\n", err.Error())
	}

	ms := float64(time.Since(startTime).Nanoseconds()) / 1000000.0
	msg(ansi.White, "\n%d triples processed in %.3f seconds (%.3f ms)\n", TriplesProcessed, ms/1000.0, ms)
	msg(ansi.White, "%d terms rewritten\n", Rewritten)
}
