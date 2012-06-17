package argo

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// A Namespace represents a namespace URI.
type Namespace string

// Common namespaces.
var (
	RDF  = NewNamespace("http://www.w3.org/1999/02/22-rdf-syntax-ns#")
	RDFS = NewNamespace("http://www.w3.org/2000/01/rdf-schema#")
	OWL  = NewNamespace("http://www.w3.org/2002/07/owl#")
	FOAF = NewNamespace("http://xmlns.com/foaf/0.1/")
	DC   = NewNamespace("http://purl.org/dc/elements/1.1/")
	DCT  = NewNamespace("http://purl.org/dc/terms/")
)

// rdf:type
var A = RDF.Get("type")

// Function NewNamespace creates and returns a new namespace with the given base URI.
func NewNamespace(base string) (ns Namespace) {
	return Namespace(base)
}

// Function Get returns a Term representing the base URI concatenated to the given local name.
// 
// The following code:
// 
//     ns := argo.NewNamespace("http://www.w3.org/1999/02/22-rdf-syntax-ns#")
//     term := ns.Get("Seq")
//     fmt.Println(fmt)
// 
// will output:
// 
//     http://www.w3.org/1999/02/22-rdf-syntax-ns#Seq
//
func (ns Namespace) Get(name string) (term Term) {
	return NewResource(string(ns) + name)
}

// Function LookupPrefix looks up the given prefix using the prefix.cc service and returns its
// namespace URI.
func LookupPrefix(prefix string) (uri string, err error) {
	reqURL := fmt.Sprintf("http://prefix.cc/%s.file.txt", prefix)

	resp, err := http.Get(reqURL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		dataBuffer := make([]byte, 1024)
		_, err := resp.Body.Read(dataBuffer)
		if err != nil {
			return "", err
		}

		data := strings.Trim(string(dataBuffer), " \r\n\x00")
		parts := strings.Split(data, "\t")
		return parts[1], nil
	}

	return "", errors.New(fmt.Sprintf("HTTP request returned status %d", resp.StatusCode))
}
