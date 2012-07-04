package sparql

import (
	"encoding/xml"
	"fmt"
	"github.com/kierdavis/argo"
	"io"
)

var (
	sparqlNS = "http://www.w3.org/2005/sparql-results#"

	sparqlSparql   = xml.Name{sparqlNS, "sparql"}
	sparqlHead     = xml.Name{sparqlNS, "head"}
	sparqlVariable = xml.Name{sparqlNS, "variable"}
	sparqlResults  = xml.Name{sparqlNS, "results"}
	sparqlBoolean  = xml.Name{sparqlNS, "boolean"}
	sparqlLink     = xml.Name{sparqlNS, "link"}
	sparqlResult   = xml.Name{sparqlNS, "result"}
	sparqlBinding  = xml.Name{sparqlNS, "binding"}
	sparqlBnode    = xml.Name{sparqlNS, "bnode"}
	sparqlUri      = xml.Name{sparqlNS, "uri"}
	sparqlLiteral  = xml.Name{sparqlNS, "literal"}

	xmlLang = xml.Name{"xml", "lang"}
)

type stateFunc func(*ResultParser, xml.Token) (stateFunc, error)

type SelectResult map[string]argo.Term

type ResultParser struct {
	decoder  *xml.Decoder
	onFinish func()
	state    stateFunc

	// Header info
	vars     []string
	linkURIs []string

	// ASK result
	boolResult bool

	// SELECT result
	currentResult   SelectResult
	currentBinding  string
	literalLanguage string
	literalDatatype string
	results         chan SelectResult
	errChan         chan error
	done            chan struct{}
}

func newResultParser(r io.Reader, onFinish func()) (l *ResultParser) {
	l = &ResultParser{
		decoder:  xml.NewDecoder(r),
		onFinish: onFinish,
		state:    parseTop,
		vars:     make([]string, 0),
		linkURIs: make([]string, 0),
		results:  make(chan SelectResult),
		errChan:  make(chan error, 1),
		done:     make(chan struct{}),
	}

	go l.process()

	return l
}

func (l *ResultParser) Vars() (vars []string) {
	return l.vars
}

func (l *ResultParser) LinkURIs() (linkURIs []string) {
	return l.linkURIs
}

func (l *ResultParser) Wait() {
	<-l.done
}

func (l *ResultParser) IsDone() (ok bool) {
	select {
	case <-l.done: // Channel is closed
		return true

	default: // Channel is still open
		return false
	}

	return false
}

func (l *ResultParser) Error() (err error) {
	return <-l.errChan
}

func (l *ResultParser) IterResults() (ch chan SelectResult) {
	return l.results
}

func (l *ResultParser) FetchResult() (result SelectResult) {
	return <-l.results
}

func (l *ResultParser) ParseInto(v interface{}) {

}

func (l *ResultParser) FetchAll() (results []SelectResult) {
	results = make([]SelectResult, 0)

	for result := range l.results {
		results = append(results, result)
	}

	return results
}

func (l *ResultParser) process() {
	defer func() {
		close(l.results)
		close(l.errChan)
		close(l.done)

		if l.onFinish != nil {
			l.onFinish()
		}
	}()

	for {
		err := l.processToken()
		if err == io.EOF {
			return
		}

		if err != nil {
			l.errChan <- err
			return
		}
	}
}

func (l *ResultParser) processToken() (err error) {
	itok, err := l.decoder.Token()
	if err != nil {
		return err
	}

	newState, err := l.state(l, itok)
	if err != nil {
		return err
	}

	if newState == nil {
		return io.EOF
	}

	l.state = newState

	return nil
}

func parseTop(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.ProcInst:
		return parseTop, nil

	case xml.CharData, xml.Comment:
		return parseTop, nil

	case xml.StartElement:
		if tok.Name != sparqlSparql {
			return nil, fmt.Errorf("Expected <sparql> element at top level")
		}

		return parseSparql, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseTop", itok)
}

func parseSparql(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseSparql, nil

	case xml.StartElement:
		if tok.Name != sparqlHead {
			return nil, fmt.Errorf("Expected <head> element inside <sparql>")
		}

		return parseHead, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseSparql", itok)
}

func parseSparql2(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseSparql2, nil

	case xml.StartElement:
		switch tok.Name {
		case sparqlResults:
			return parseResults, nil

		case sparqlBoolean:
			return parseBoolean, nil

		default:
			return nil, fmt.Errorf("Expected <results> or <boolean> element inside <sparql>")
		}
	}

	return nil, fmt.Errorf("Unexpected %T in parseSparql2", itok)
}

func parseHead(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseHead, nil

	case xml.StartElement:
		switch tok.Name {
		case sparqlVariable:
			l.vars = append(l.vars, getAttr(tok.Attr, xml.Name{"", "name"}))
			return parseHead, nil

		case sparqlLink:
			l.linkURIs = append(l.linkURIs, getAttr(tok.Attr, xml.Name{"", "href"}))
			return parseHead, nil

		default:
			return nil, fmt.Errorf("Expected <variable> or <link> in <head>, not <%s>", tok.Name.Local)
		}

	case xml.EndElement:
		switch tok.Name {
		case sparqlVariable, sparqlLink:
			return parseHead, nil

		case sparqlHead:
			return parseSparql2, nil
		}
	}

	return nil, fmt.Errorf("Unexpected %T in parseHead", itok)
}

func parseResults(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseResults, nil

	case xml.StartElement:
		if tok.Name != sparqlResult {
			return nil, fmt.Errorf("Expected <result> element inside <results>")
		}

		l.currentResult = make(SelectResult)

		return parseResult, nil

	case xml.EndElement: // </results>
		return parseFinish, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseResults", itok)
}

func parseResult(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseResult, nil

	case xml.StartElement:
		if tok.Name != sparqlBinding {
			return nil, fmt.Errorf("Expected <binding> element inside <result>")
		}

		l.currentBinding = getAttr(tok.Attr, xml.Name{"", "name"})

		return parseBinding, nil

	case xml.EndElement: // </result>
		l.results <- l.currentResult
		l.currentResult = nil

		return parseResults, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseResult", itok)
}

func parseBinding(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseBinding, nil

	case xml.StartElement:
		switch tok.Name {
		case sparqlBnode:
			return parseBnode, nil

		case sparqlUri:
			return parseUri, nil

		case sparqlLiteral:
			l.literalLanguage = getAttr(tok.Attr, xmlLang)
			l.literalDatatype = getAttr(tok.Attr, xml.Name{"", "datatype"})

			return parseLiteral, nil

		default:
			return nil, fmt.Errorf("Expected <bnode>, <uri> or <literal> element inside <result>")
		}

	case xml.EndElement: // </binding>
		return parseResult, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBinding", itok)
}

func parseBnode(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseBnode, nil

	case xml.CharData:
		l.currentResult[l.currentBinding] = argo.NewBlankNode(string(tok))
		l.currentBinding = ""

		return parseBnode, nil

	case xml.EndElement: // </bnode>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBnode", itok)
}

func parseUri(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseUri, nil

	case xml.CharData:
		l.currentResult[l.currentBinding] = argo.NewResource(string(tok))
		l.currentBinding = ""

		return parseUri, nil

	case xml.EndElement: // </uri>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseUri", itok)
}

func parseLiteral(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseLiteral, nil

	case xml.CharData:
		var datatype argo.Term

		if l.literalDatatype != "" {
			datatype = argo.NewResource(l.literalDatatype)
		}

		l.currentResult[l.currentBinding] = argo.NewLiteralWithLanguageAndDatatype(string(tok), l.literalLanguage, datatype)
		l.currentBinding = ""

		return parseLiteral, nil

	case xml.EndElement: // </literal>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseLiteral", itok)
}

func parseBoolean(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseTop, nil

	case xml.CharData:
		switch string(tok) {
		case "true":
			l.boolResult = true

		case "false":
			l.boolResult = false

		default:
			return nil, fmt.Errorf("Invalid value for <boolean>: %s", string(tok))
		}

		return parseBoolean, nil

	case xml.EndElement: // </boolean>
		return parseFinish, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBoolean", itok)
}

func parseFinish(l *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch itok.(type) {
	case xml.CharData, xml.Comment:
		return parseFinish, nil

	case xml.EndElement: // </sparql>
		return nil, nil // Done
	}

	return nil, fmt.Errorf("Unexpected %T in parseFinish", itok)
}

func getAttr(attrs []xml.Attr, name xml.Name) (value string) {
	for _, attr := range attrs {
		if name == attr.Name {
			return attr.Value
		}
	}

	return ""
}

/*
var stateNames = map[stateFunc]string{
	parseTop:     "parseTop",
	parseSparql:  "parseSparql",
	parseSparql2: "parseSparql2",
	parseHead:    "parseHead",
	parseResults: "parseResults",
	parseResult:  "parseResult",
	parseBinding: "parseBinding",
	parseBnode:   "parseBnode",
	parseUri:     "parseUri",
	parseLiteral: "parseLiteral",
	parseBoolean: "parseBoolean",
	parseFinish:  "parseFinish",
	nil:          "nil",
}
*/
