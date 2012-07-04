/*
	Copyright (c) 2012 Kier Davis

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
	associated documentation files (the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge, publish, distribute,
	sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial
	portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
	NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
	NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
	OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
	CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

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
	XSD  = NewNamespace("http://www.w3.org/2001/XMLSchema#")
)

// RDF vocab elements that are used internally by the library.
var (
	A     = RDF.Get("type")
	First = RDF.Get("first")
	Rest  = RDF.Get("rest")
	Nil   = RDF.Get("nil")
)

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
//     fmt.Println(term.String())
// 
// will output:
// 
//     <http://www.w3.org/1999/02/22-rdf-syntax-ns#Seq>
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
