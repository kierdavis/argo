package sparql

import (
	"bytes"
	"fmt"
	"github.com/kierdavis/argo"
	"net/http"
	"net/url"
	"strings"
)

type GraphStoreService struct {
	EndpointURI string
	Debug       bool
}

func NewGraphStoreService(endpointURI string) (service GraphStoreService) {
	return GraphStoreService{
		EndpointURI: endpointURI,
	}
}

func (service GraphStoreService) ActionURI(graphURI string) (actionURI string) {
	var params string

	if graphURI == "" {
		params = url.Values{
			"default": {""},
		}.Encode()

	} else {
		params = url.Values{
			"graph": {graphURI},
		}.Encode()
	}

	return service.EndpointURI + "?" + params
}

func (service GraphStoreService) Get(graphURI string) (graph *argo.Graph, err error) {
	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("GET", actionURI)
	}

	req, err := http.NewRequest("GET", actionURI, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/rdf+xml")

	resp, err := EnsureOK(http.DefaultClient.Do(req))
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

func (service GraphStoreService) Put(graphURI string, graph *argo.Graph) (err error) {
	buf := new(bytes.Buffer)
	err = graph.Serialize(argo.SerializeRDFXML, buf)
	if err != nil {
		return err
	}

	//fmt.Println(string(buf.Bytes()))

	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("PUT", actionURI)
	}

	req, err := http.NewRequest("PUT", actionURI, buf)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/rdf+xml")

	_, err = DropBody(EnsureOK(http.DefaultClient.Do(req)))
	if err != nil {
		return err
	}

	return nil
}

func (service GraphStoreService) Delete(graphURI string) (err error) {
	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("DELETE", actionURI)
	}

	req, err := http.NewRequest("DELETE", actionURI, nil)
	if err != nil {
		return err
	}

	_, err = DropBody(EnsureOK(http.DefaultClient.Do(req)))
	if err != nil {
		return err
	}

	return nil
}

func (service GraphStoreService) Post(graphURI string, graph *argo.Graph) (err error) {
	buf := new(bytes.Buffer)
	err = graph.Serialize(argo.SerializeRDFXML, buf)
	if err != nil {
		return err
	}

	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("POST", actionURI)
	}

	req, err := http.NewRequest("POST", actionURI, buf)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/rdf+xml")

	_, err = DropBody(EnsureOK(http.DefaultClient.Do(req)))
	if err != nil {
		return err
	}

	return nil
}

func (service GraphStoreService) Head(graphURI string) (err error) {
	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("HEAD", actionURI)
	}

	req, err := http.NewRequest("HEAD", actionURI, nil)
	if err != nil {
		return err
	}

	_, err = DropBody(EnsureOK(http.DefaultClient.Do(req)))
	if err != nil {
		return err
	}

	return nil
}

func (service GraphStoreService) Patch(graphURI string, updateQuery string) (err error) {
	actionURI := service.ActionURI(graphURI)
	if service.Debug {
		fmt.Println("PATCH", actionURI)
	}

	req, err := http.NewRequest("PATCH", actionURI, strings.NewReader(updateQuery))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/sparql-update")

	_, err = DropBody(EnsureOK(http.DefaultClient.Do(req)))
	if err != nil {
		return err
	}

	return nil
}
