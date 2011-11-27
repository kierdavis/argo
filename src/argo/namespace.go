package argo

import (
	"fmt"
	"http"
	"os"
	"strings"
)

type Namespace struct {
	base string
}

func CreateNamespace(base string) (ns *Namespace) {
	return &Namespace{base: base}
}

func (ns *Namespace) Base() (base string) {
	return ns.base
}

func (ns *Namespace) SetBase(base string) {
	ns.base = base
}

func (ns *Namespace) Get(name string) (term *Term) {
	return CreateResource(strings.Join([]string{ns.base, name}, ""))
}

func Lookup(prefix string) (uri string, err os.Error) {
	reqURL := fmt.Sprintf("http://prefix.cc/%s.file.txt", prefix)

	resp, finalURL, err := http.Get(reqURL)
	if err != nil {return "", err}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		dataBuffer := new([1024]byte)[:]
		n, err := resp.Body.Read(dataBuffer)
		if err != nil {return "", err}

		data := strings.Trim(string(dataBuffer), " \r\n\x00")
		parts := strings.Split(data, "\t", 2)
		return parts[1], nil
	}
	
	return "", os.ErrorString(fmt.Sprintf("HTTP request returned status %d", resp.StatusCode))
}

var RDF = CreateNamespace("http://www.w3.org/1999/02/22-rdf-syntax-ns#")
var RDFS = CreateNamespace("http://www.w3.org/2000/01/rdf-schema#")
var OWL = CreateNamespace("http://www.w3.org/2002/07/owl#")
var FOAF = CreateNamespace("http://xmlns.com/foaf/0.1/")
var DC = CreateNamespace("http://purl.org/dc/elements/1.1/")
var DCT = CreateNamespace("http://purl.org/dc/terms/")
var A = RDF.Get("type")
