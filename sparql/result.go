package sparql

import (
	"encoding/xml"
	"fmt"
	"github.com/kierdavis/argo"
	"io"
	"reflect"
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

type StructuredResultParser struct {
	rp     *ResultParser
	value  reflect.Value
	rename map[string]string
}

func NewStructuredResultParser(rp *ResultParser, v interface{}) (srp *StructuredResultParser, err error) {
	value := reflect.ValueOf(v)

	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}

	t := value.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Invalid type: expected a struct, got a %s", value.Type().Kind())
	}

	rename := make(map[string]string) // Binding -> struct field

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("sparql")
		if tag != "" {
			rename[tag] = field.Name
		}
	}

	return &StructuredResultParser{
		rp:     rp,
		value:  value,
		rename: rename,
	}, nil
}

func (srp *StructuredResultParser) Read() (err error) {
	result := srp.rp.ReadResult()
	if result == nil {
		return io.EOF
	}

	for key, value := range result {
		if key == "" {
			continue
		}

		renamed, isRenamed := srp.rename[key]
		if isRenamed {
			key = renamed
		}

		field := srp.value.FieldByName(key)

		if !field.IsValid() {
			return fmt.Errorf("Could not find a destination field for binding '%s' (try using a struct tag `sparql:\"BINDING_NAME\"`)", key)
		}

		field.Set(reflect.ValueOf(value))
	}

	return nil
}

type ResultParser struct {
	decoder  *xml.Decoder
	onFinish func()
	state    stateFunc

	// Header info
	vars     []string
	linkURIs []string

	// Signals
	done       chan struct{}
	headerDone chan struct{}

	// ASK result
	boolResult bool

	// SELECT result
	currentResult   SelectResult
	currentBinding  string
	literalLanguage string
	literalDatatype string
	results         chan SelectResult
	errChan         chan error
}

func newResultParser(r io.Reader, onFinish func()) (rp *ResultParser) {
	rp = &ResultParser{
		decoder:    xml.NewDecoder(r),
		onFinish:   onFinish,
		state:      parseTop,
		vars:       make([]string, 0),
		linkURIs:   make([]string, 0),
		done:       make(chan struct{}),
		headerDone: make(chan struct{}),
		results:    make(chan SelectResult),
		errChan:    make(chan error, 1),
	}

	go rp.process()

	return rp
}

func (rp *ResultParser) Vars() (vars []string) {
	rp.WaitUntilHeaderDone()
	return rp.vars
}

func (rp *ResultParser) LinkURIs() (linkURIs []string) {
	rp.WaitUntilHeaderDone()
	return rp.linkURIs
}

func (rp *ResultParser) WaitUntilDone() {
	<-rp.done
}

func (rp *ResultParser) WaitUntilHeaderDone() {
	<-rp.headerDone
}

func (rp *ResultParser) IsDone() (ok bool) {
	select {
	case <-rp.done: // Channel is closed
		return true

	default: // Channel is still open
		return false
	}

	return false
}

func (rp *ResultParser) IsHeaderDone() (ok bool) {
	select {
	case <-rp.headerDone: // Channel is closed
		return true

	default: // Channel is still open
		return false
	}

	return false
}

func (rp *ResultParser) Error() (err error) {
	return <-rp.errChan
}

func (rp *ResultParser) ResultChan() (ch chan SelectResult) {
	return rp.results
}

func (rp *ResultParser) ReadResult() (result SelectResult) {
	return <-rp.results
}

func (rp *ResultParser) ReadAll() (results []SelectResult) {
	results = make([]SelectResult, 0)

	for result := range rp.results {
		results = append(results, result)
	}

	return results
}

func (rp *ResultParser) process() {
	defer func() {
		close(rp.results)
		close(rp.errChan)
		close(rp.done)

		if rp.onFinish != nil {
			rp.onFinish()
		}
	}()

	for {
		err := rp.processToken()
		if err == io.EOF {
			return
		}

		if err != nil {
			rp.errChan <- err
			return
		}
	}
}

func (rp *ResultParser) processToken() (err error) {
	itok, err := rp.decoder.Token()
	if err != nil {
		return err
	}

	newState, err := rp.state(rp, itok)
	if err != nil {
		return err
	}

	if newState == nil {
		return io.EOF
	}

	rp.state = newState

	return nil
}

func parseTop(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
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

func parseSparql(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
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

func parseSparql2(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
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

func parseHead(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseHead, nil

	case xml.StartElement:
		switch tok.Name {
		case sparqlVariable:
			rp.vars = append(rp.vars, getAttr(tok.Attr, xml.Name{"", "name"}))
			return parseHead, nil

		case sparqlLink:
			rp.linkURIs = append(rp.linkURIs, getAttr(tok.Attr, xml.Name{"", "href"}))
			return parseHead, nil

		default:
			return nil, fmt.Errorf("Expected <variable> or <link> in <head>, not <%s>", tok.Name.Local)
		}

	case xml.EndElement:
		switch tok.Name {
		case sparqlVariable, sparqlLink:
			return parseHead, nil

		case sparqlHead:
			close(rp.headerDone)
			return parseSparql2, nil
		}
	}

	return nil, fmt.Errorf("Unexpected %T in parseHead", itok)
}

func parseResults(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseResults, nil

	case xml.StartElement:
		if tok.Name != sparqlResult {
			return nil, fmt.Errorf("Expected <result> element inside <results>")
		}

		rp.currentResult = make(SelectResult)

		return parseResult, nil

	case xml.EndElement: // </results>
		return parseFinish, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseResults", itok)
}

func parseResult(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.CharData, xml.Comment:
		return parseResult, nil

	case xml.StartElement:
		if tok.Name != sparqlBinding {
			return nil, fmt.Errorf("Expected <binding> element inside <result>")
		}

		rp.currentBinding = getAttr(tok.Attr, xml.Name{"", "name"})

		return parseBinding, nil

	case xml.EndElement: // </result>
		rp.results <- rp.currentResult
		rp.currentResult = nil

		return parseResults, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseResult", itok)
}

func parseBinding(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
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
			rp.literalLanguage = getAttr(tok.Attr, xmlLang)
			rp.literalDatatype = getAttr(tok.Attr, xml.Name{"", "datatype"})

			return parseLiteral, nil

		default:
			return nil, fmt.Errorf("Expected <bnode>, <uri> or <literal> element inside <result>")
		}

	case xml.EndElement: // </binding>
		return parseResult, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBinding", itok)
}

func parseBnode(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseBnode, nil

	case xml.CharData:
		rp.currentResult[rp.currentBinding] = argo.NewBlankNode(string(tok))
		rp.currentBinding = ""

		return parseBnode, nil

	case xml.EndElement: // </bnode>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBnode", itok)
}

func parseUri(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseUri, nil

	case xml.CharData:
		rp.currentResult[rp.currentBinding] = argo.NewResource(string(tok))
		rp.currentBinding = ""

		return parseUri, nil

	case xml.EndElement: // </uri>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseUri", itok)
}

func parseLiteral(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseLiteral, nil

	case xml.CharData:
		var datatype argo.Term

		if rp.literalDatatype != "" {
			datatype = argo.NewResource(rp.literalDatatype)
		}

		rp.currentResult[rp.currentBinding] = argo.NewLiteralWithLanguageAndDatatype(string(tok), rp.literalLanguage, datatype)
		rp.currentBinding = ""

		return parseLiteral, nil

	case xml.EndElement: // </literal>
		return parseBinding, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseLiteral", itok)
}

func parseBoolean(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
	switch tok := itok.(type) {
	case xml.Comment:
		return parseTop, nil

	case xml.CharData:
		switch string(tok) {
		case "true":
			rp.boolResult = true

		case "false":
			rp.boolResult = false

		default:
			return nil, fmt.Errorf("Invalid value for <boolean>: %s", string(tok))
		}

		return parseBoolean, nil

	case xml.EndElement: // </boolean>
		return parseFinish, nil
	}

	return nil, fmt.Errorf("Unexpected %T in parseBoolean", itok)
}

func parseFinish(rp *ResultParser, itok xml.Token) (newState stateFunc, err error) {
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
