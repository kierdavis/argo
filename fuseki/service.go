package fuseki

type Service struct {
	BaseURI string
}

func NewService(baseURI string) (service Service) {
	if baseURI[len(baseURI)-1] == '/' {
		baseURI = baseURI[:len(baseURI)-1]
	}

	return Service{
		BaseURI: baseURI,
	}
}

func (service Service) Dataset(name string) (dataset Dataset) {
	return NewDataset(service.BaseURI + "/" + name)
}
