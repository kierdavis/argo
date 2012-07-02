package main

import (
	"github.com/kierdavis/argo"
	"github.com/kierdavis/argo/squirtle"
	"github.com/kierdavis/go/ansi"
	"github.com/kierdavis/go/argparse"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

var FormatNames = []string{
	"ntriples",
	"rdfxml",
	"squirtle",
}

var (
	infoStyle = ansi.Attribute{FG: ansi.Blue}
	errStyle  = ansi.Attribute{FG: ansi.Red, Attr: ansi.Bold}
)

type Args struct {
	OutFile      string
	URLs         []string
	Files        []string
	OutputFormat string
	InputFormat  string
	StdinFormat  string
}

func pipe(src chan *argo.Triple, dest chan *argo.Triple) {
	for triple := range src {
		dest <- triple
	}
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

func determineSerializerByExtension(path string) (serializer argo.Serializer) {
	if strings.HasSuffix(path, ".nt") || strings.HasSuffix(path, ".txt") {
		return argo.SerializeNTriples
	}

	return argo.SerializeRDFXML
}

func read(output chan *argo.Triple, prefixMap map[string]string, args *Args) {
	// Concurrent loading, gives a minimal speed gain:

	var wg sync.WaitGroup

	for _, url := range args.URLs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				ansi.Fprintf(os.Stderr, errStyle, "Error when preparing to fetch '%s': %s\n", url, err.Error())
				return
			}

			parser, mimetype := determineParserByExtension(url)
			req.Header.Add("Accept", mimetype)

			ansi.Printf(infoStyle, "Fetching '%s'...\n", url)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				ansi.Fprintf(os.Stderr, errStyle, "Error when fetching '%s': %s\n", url, err.Error())
				return
			}
			defer resp.Body.Close()

			ansi.Printf(infoStyle, "Parsing '%s'...\n", url)
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
				ansi.Fprintf(os.Stderr, errStyle, "Error when parsing '%s': %s\n", url, err.Error())
				return
			}

			ansi.Printf(infoStyle, "Parsed '%s' successfully!\n", url)
		}()
	}

	for _, file := range args.Files {
		wg.Add(1)

		if file == "-" {
			go func() {
				defer wg.Done()

				ansi.Printf(infoStyle, "Parsing standard input...\n")
				tripleChan := make(chan *argo.Triple)
				errChan := make(chan error)
				go argo.ParseRDFXML(os.Stdin, tripleChan, errChan, prefixMap)

				wg.Add(1)
				go func() {
					pipe(tripleChan, output)
					wg.Done()
				}()

				err := <-errChan
				if err != nil {
					ansi.Fprintf(os.Stderr, errStyle, "Error when parsing standard input: %s\n", err.Error())
					return
				}

				ansi.Printf(infoStyle, "Parsed standard input successfully!\n")
			}()

		} else {
			go func() {
				defer wg.Done()

				parser, _ := determineParserByExtension(file)

				f, err := os.Open(file)
				if err != nil {
					ansi.Fprintf(os.Stderr, errStyle, "Error when opening '%s': %s\n", file, err.Error())
					return
				}
				defer f.Close()

				ansi.Printf(infoStyle, "Parsing '%s'...\n", file)
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
					ansi.Fprintf(os.Stderr, errStyle, "Error when parsing '%s': %s\n", file, err.Error())
					return
				}

				ansi.Printf(infoStyle, "Parsed '%s' successfully!\n", file)
			}()
		}
	}

	wg.Wait()
	close(output)
}

func main() {
	args := &Args{
		OutFile: "-",
	}

	p := argparse.New("A tool for manipulating RDF files.")
	p.Option('o', "output", "OutFile", 1, argparse.Store, "FILENAME", "The file to write output to. Default: standard output.")
	p.Option('u', "url", "URLs", 1, argparse.Append, "URL", "A URL to download from and add to the graph. Can be used multiple times. Default: no URLs will be downloaded.")
	p.Option('O', "output-format", "OutputFormat", 1, argparse.Choice(argparse.Store, FormatNames), "FORMAT", "The format to write output to. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('I', "input-format", "InputFormat", 1, argparse.Choice(argparse.Store, FormatNames), "FORMAT", "The format to parse all input sources as. Default: determine by the file extension, or fall back to rdfxml if unavailable.")
	p.Option('i', "stdin-format", "StdinFormat", 1, argparse.Choice(argparse.Store, FormatNames), "FORMAT", "The format to parse stdin as. The formats for all other sources (files and URLs) are still determined by their file extensions. Default: rdfxml.")
	p.Argument("Files", argparse.ZeroOrMore, argparse.Store, "filename", "Files to parse and add to the graph.")
	err := p.Parse(args)

	if err != nil {
		ansi.Fprintf(os.Stderr, errStyle, "Error when parsing arguments: %s\n", err.Error())
		os.Exit(1)
	}

	var output io.Writer
	var serializer argo.Serializer

	if args.OutFile == "-" {
		output = os.Stdout
		serializer = argo.SerializeRDFXML

	} else {
		output, err = os.Open(args.OutFile)
		if err != nil {
			ansi.Fprintf(os.Stderr, errStyle, "Error when opening '%s': %s\n", args.OutFile, err.Error())
			os.Exit(1)
		}

		serializer = determineSerializerByExtension(args.OutFile)
	}

	tripleChan := make(chan *argo.Triple)
	graph := argo.NewGraph(argo.NewListStore())

	go read(tripleChan, graph.Prefixes, args)
	graph.LoadFromChannel(tripleChan)

	err = graph.Serialize(serializer, output)

	if err != nil {
		ansi.Fprintf(os.Stderr, errStyle, "Error when serializing: %s\n", args.OutFile, err.Error())
		os.Exit(1)
	}
}
