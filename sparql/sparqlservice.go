package sparql

import (
	"fmt"
	"github.com/kierdavis/argo"
	"net/http"
	"net/url"
	"strings"
)

type SparqlService struct {
	EndpointURI string
	Debug       bool
}

func NewSparqlService(endpointURI string) (service SparqlService) {
	return SparqlService{
		EndpointURI: endpointURI,
	}
}

func (service SparqlService) do(form url.Values, accept string) (resp *http.Response, err error) {
	payload := form.Encode()

	if service.Debug {
		fmt.Println("POST", service.EndpointURI, payload)
	}

	req, err := http.NewRequest("POST", service.EndpointURI, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if accept != "" {
		req.Header.Add("Accept", accept)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err = EnsureOK(http.DefaultClient.Do(req))

	return resp, err
}

func (service SparqlService) Select(query string) (results *ResultParser, err error) {
	resp, err := service.do(url.Values{"query": {query}}, "application/sparql-results+xml")
	if err != nil {
		return nil, err
	}

	onFinish := func() {
		resp.Body.Close()
	}

	return newResultParser(resp.Body, onFinish), nil
}

func (service SparqlService) Ask(query string) (result bool, err error) {
	resp, err := service.do(url.Values{"query": {query}}, "application/sparql-results+xml")
	if err != nil {
		return false, err
	}

	onFinish := func() {
		resp.Body.Close()
	}

	l := newResultParser(resp.Body, onFinish)
	l.WaitUntilDone()

	return l.boolResult, l.Error()
}

func (service SparqlService) Graph(query string) (graph *argo.Graph, err error) {
	resp, err := service.do(url.Values{"query": {query}}, "application/rdf+xml")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	graph = argo.NewGraph(argo.NewListStore())
	err = graph.Parse(argo.ParseRDFXML, resp.Body)
	if err != nil {
		return nil, err
	}

	return graph, nil
}

func (service SparqlService) Update(query string) (err error) {
	_, err = DropBody(service.do(url.Values{"update": {query}}, ""))
	if err != nil {
		return err
	}

	return nil
}
