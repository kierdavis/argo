package argo_kasabi

import (
	"fmt"
	"http"
	"json"
	"os"
	"argo"
)

type Dataset struct {
	apiKey string
	name string
	verbosity int8
}

type SparqlSelectResult struct {
	query string
	vars []string
	results []map[string]*argo.Term
}

func OpenDataset(name string, apiKey string, verbosity int8) (dataset *Dataset) {
	return &Dataset{name: name, apiKey: apiKey, verbosity: verbosity}
}

func (dataset *Dataset) Name() (name string) {
	return dataset.name
}

func (dataset *Dataset) SetName(name string) {
	dataset.name = name
}

func (dataset *Dataset) ApiKey() (apiKey string) {
	return dataset.apiKey
}

func (dataset *Dataset) SetApiKey(apiKey string) {
	dataset.apiKey = apiKey
}

/*
func (dataset *Dataset) doRequest(url string, params url.Values) (resp *http.Response, finalURL string, err os.Error) {
	params.Add("apikey", dataset.apiKey)
	url = fmt.Sprintf("%s?%s", url, params.Encode())

	return http.Get(url)
}
*/

func (dataset *Dataset) Lookup(uri string) (graph *argo.Graph, err os.Error) {
	url := fmt.Sprintf("http://api.kasabi.com/dataset/%s/apis/lookup?apikey=%s&about=%s&output=json", dataset.name, http.URLEscape(dataset.apiKey), http.URLEscape(uri))

	if dataset.verbosity > 0 {
		fmt.Println("GET", url)
	}

	resp, finalURL, err := http.Get(url)
	if err != nil {return nil, err}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, os.ErrorString(fmt.Sprintf("HTTP request returned status %d", resp.StatusCode))
	}

	graph = argo.CreateGraphWithMemoryStore()
	graph.ReadJSON(resp.Body)
	return graph, nil
}

func (dataset *Dataset) SparqlSelect(query string) (result *SparqlSelectResult, err os.Error) {
	url := fmt.Sprintf("http://api.kasabi.com/dataset/%s/apis/sparql?apikey=%s&query=%s&output=json", dataset.name, http.URLEscape(dataset.apiKey), http.URLEscape(query))

	if dataset.verbosity > 0 {
		fmt.Println("GET", url)
	}

	resp, finalURL, err := http.Get(url)
	if err != nil {return nil, err}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, os.ErrorString(fmt.Sprintf("HTTP request returned status %d", resp.StatusCode))
	}

	data := new(map[string]map[string][]interface{})

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(data)

	result = new(SparqlSelectResult)
	result.query = query
	result.results = []map[string]*argo.Term{}
	result.vars = []string{}

	for i, v := range (*data)["head"]["vars"] {
		result.vars = append(result.vars, string(v))
	}

	for i, binding := range (*data)["results"]["bindings"] {
		m := map[string]*argo.Term{}
		b := map[string]interface{}(binding)

		for name, obj := range b {
			term, err := argo.ParseJSONTerm(map[string]string(obj))
			if err != nil {return nil, err}
			m[name] = term
		}

		result.results = append(result.results, m)
	}

	return result, nil
}
