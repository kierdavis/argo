package main

import (
	"bufio"
	"fmt"
	"github.com/kierdavis/ansi"
	"github.com/kierdavis/argo"
	"github.com/kierdavis/argo/fuseki"
	"github.com/kierdavis/argo/sparql"
	"github.com/kierdavis/argo/squirtle"
	"github.com/kierdavis/argparse"
	"io"
	"os"
	"regexp"
	"strings"
)

var PrefixRegexp = regexp.MustCompile(`^\s*[pP][rR][eE][fF][iI][xX]\s+(\w+)\s*:\s*<(.+)>\s*\.\s*`)

type Args struct {
	Endpoint       string
	UpdateEndpoint string
	UseFuseki      bool
	Debug          bool
}

func max(a, b int) (q int) {
	if a > b {
		return a
	}

	return b
}

type Table struct {
	Header []string
	Data   [][]string
	Widths []int
}

func (table *Table) SetHeader(items ...string) {
	table.Header = items
	table.Widths = make([]int, len(items))

	for i, item := range items {
		table.Widths[i] = len(item)
	}
}

func (table *Table) AddRow(items ...string) {
	for i, item := range items {
		table.Widths[i] = max(table.Widths[i], len(item))
	}

	table.Data = append(table.Data, items)
}

func (table *Table) Print() {
	table.PrintBoundary()
	table.PrintRow(table.Header, center)
	table.PrintBoundary()

	for _, row := range table.Data {
		table.PrintRow(row, leftAlign)
	}

	table.PrintBoundary()
}

func (table *Table) PrintBoundary() {
	for _, width := range table.Widths {
		fmt.Print("+-" + strings.Repeat("-", width) + "-")
	}

	fmt.Print("+\n")
}

func (table *Table) PrintRow(items []string, aligner func(string, int) string) {
	for i, item := range items {
		fmt.Print("| " + aligner(item, table.Widths[i]) + " ")
	}

	fmt.Print("|\n")
}

func die(err error, exit bool) (shouldExit bool) {
	if err != nil {
		ansi.Fprintf(os.Stderr, ansi.RedBold, "Error: %s\n", err.Error())

		if exit {
			os.Exit(1)
		}

		return true
	}

	return false
}

func trimPrefixes(line string, prefixes map[string]string) (newLine string) {
	groups := PrefixRegexp.FindStringSubmatch(line)
	if groups != nil {
		prefixes[groups[1]] = groups[2]
		return trimPrefixes(line[len(groups[0]):], prefixes)
	}

	return line
}

func center(s string, width int) (res string) {
	spacing := width - len(s)
	if spacing < 0 {
		spacing = 0
	}

	halfSpacing := spacing / 2
	return strings.Repeat(" ", halfSpacing) + s + strings.Repeat(" ", spacing-halfSpacing)
}

func leftAlign(s string, width int) (res string) {
	spacing := width - len(s)
	if spacing < 0 {
		spacing = 0
	}

	return s + strings.Repeat(" ", spacing)
}

func update(a, b map[string]string) {
	for k, v := range b {
		a[k] = v
	}
}

func updateRev(a, b map[string]string) {
	for k, v := range b {
		a[v] = k
	}
}

func main() {
	p := argparse.New("A SPARQL query & update client")
	p.Argument("Endpoint", 1, argparse.Store, "endpoint_uri", "The SPARQL endpoint URI. It is used for all query operations, and update operations when -u is not specified.")
	p.Option('u', "update-endpoint", "UpdateEndpoint", 1, argparse.Store, "URI", "An alternative endpoint URI that is only used for SPARQL update operations. Default: use the query endpoint URI.")
	p.Option('f', "fuseki", "UseFuseki", 0, argparse.StoreConst(true), "", "Interpret endpoint_uri as the URI of a Fuseki dataset, and then use its query and update services as the corresponding endpoints for the session.")
	p.Option('d', "debug", "Debug", 0, argparse.StoreConst(true), "", "Show debug info.")

	args := &Args{}
	err := p.Parse(args)

	if err != nil {
		if cmdLineErr, ok := err.(argparse.CommandLineError); ok {
			ansi.Fprintln(os.Stderr, ansi.RedBold, string(cmdLineErr))
			p.Help()
			os.Exit(2)

		} else {
			die(err, true)
		}
	}

	var queryService, updateService sparql.SparqlService

	if args.UseFuseki {
		dataset := fuseki.NewDataset(args.Endpoint)
		queryService = dataset.QueryService()
		updateService = dataset.UpdateService()

	} else {
		queryService = sparql.NewSparqlService(args.Endpoint)

		if args.UpdateEndpoint != "" {
			updateService = sparql.NewSparqlService(args.UpdateEndpoint)

		} else {
			updateService = queryService
		}
	}

	queryService.Debug = args.Debug
	updateService.Debug = args.Debug

	stdinReader := bufio.NewReader(os.Stdin)
	prefixes := make(map[string]string) // Prefix -> Base URI
	serializer := argo.SerializeRDFXML

mainloop:
	for {
		fmt.Print("> ")

		line, err := stdinReader.ReadString('\n')
		if err == io.EOF {
			return
		}

		if die(err, false) {
			continue mainloop
		}

		line = trimPrefixes(line[:len(line)-1], prefixes)
		line = strings.Trim(line, " \r\n\t")

		if line == "" {
			continue mainloop
		}

		verb := line
		spacePos := strings.IndexRune(line, ' ')
		if spacePos >= 0 {
			verb = line[:spacePos]
		}

		switch strings.ToUpper(verb) {
		case "SELECT":
			rp, err := queryService.Select(line)
			if die(err, false) {
				continue mainloop
			}

			vars := rp.Vars()

			var table Table
			table.SetHeader(vars...)

			for result := range rp.ResultChan() {
				fields := make([]string, len(vars))

				for i, v := range vars {
					fields[i] = result[v].String()
				}

				table.AddRow(fields...)
			}

			ansi.AttrOn(ansi.Yellow)
			table.Print()
			ansi.AttrOff(ansi.Yellow)

		case "ASK":
			result, err := queryService.Ask(line)
			if die(err, false) {
				continue mainloop
			}

			ansi.Printf(ansi.Magenta, "Result: %t\n", result)

		case "CONSTRUCT", "DESCRIBE":
			graph, err := queryService.Graph(line)
			if die(err, false) {
				continue mainloop
			}

			updateRev(graph.Prefixes, prefixes)

			ansi.AttrOn(ansi.Cyan)
			graph.Serialize(serializer, os.Stdout)
			ansi.AttrOff(ansi.Cyan)

		case "INSERT", "DELETE", "LOAD", "CLEAR", "CREATE", "DROP", "COPY", "MOVE", "ADD":
			err := updateService.Update(line)
			if die(err, false) {
				continue mainloop
			}

			ansi.Println(ansi.GreenBold, "OK")

		case "FORMAT":
			format := strings.ToLower(line[spacePos+1:])

			switch format {
			case "xml", "rdfxml":
				serializer = argo.SerializeRDFXML

			case "nt", "ntriples":
				serializer = argo.SerializeNTriples

			case "ttl", "turtle":
				serializer = argo.SerializeTurtle

			case "sq", "squirtle":
				serializer = squirtle.SerializeSquirtle

			default:
				ansi.Fprintf(os.Stderr, ansi.RedBold, "Invalid format: %s\n", format)
				continue mainloop
			}

		default:
			ansi.Fprintf(os.Stderr, ansi.RedBold, "Invalid command: %s\n", verb)
		}
	}
}
