package fuseki

import (
	"github.com/kierdavis/argo/sparql"
)

type Dataset struct {
	BaseURI string
}

func NewDataset(baseURI string) (dataset Dataset) {
	if baseURI[len(baseURI)-1] == '/' {
		baseURI = baseURI[:len(baseURI)-1]
	}

	return Dataset{
		BaseURI: baseURI,
	}
}

func (dataset Dataset) QueryEndpoint() (uri string) {
	return dataset.BaseURI + "/query"
}

func (dataset Dataset) UpdateEndpoint() (uri string) {
	return dataset.BaseURI + "/update"
}

func (dataset Dataset) GraphStoreEndpoint() (uri string) {
	return dataset.BaseURI + "/data"
}

func (dataset Dataset) UploadEndpoint() (uri string) {
	return dataset.BaseURI + "/upload"
}

func (dataset Dataset) QueryService() (service sparql.SparqlService) {
	return sparql.NewSparqlService(dataset.QueryEndpoint())
}

func (dataset Dataset) UpdateService() (service sparql.SparqlService) {
	return sparql.NewSparqlService(dataset.UpdateEndpoint())
}

func (dataset Dataset) GraphStoreService() (service sparql.GraphStoreService) {
	return sparql.NewGraphStoreService(dataset.GraphStoreEndpoint())
}
