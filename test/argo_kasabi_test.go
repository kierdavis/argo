package main

import (
	"strings"
	"os"
	//"argo"
	"argo_kasabi"
)

func GetApiKey() (apiKey string, err os.Error) {
	reader, err := os.Open("apikey.txt", os.O_RDONLY, 0666)
	if err != nil {return "", err}
	buffer := new([1024]byte)[:]
	n, err := reader.Read(buffer)
	if err != nil {return "", err}
	apiKey = strings.Trim(string(buffer), " \n\t\x00")
	err = reader.Close()
	if err != nil {return "", err}
	
	return apiKey, nil
}

func main() {
	apiKey, err := GetApiKey()
	if err != nil {panic(err)}

	dataset := argo_kasabi.OpenDataset("icdb", apiKey, 1)

	graph, err := dataset.Lookup("http://data.kasabi.com/dataset/icdb/ic/ic555")
	if err != nil {panic(err)}
	graph.WriteXML(os.Stdout)

	/*
	result, err := dataset.SparqlSelect("SELECT ?p ?o WHERE {<http://data.kasabi.com/dataset/icdb/ic/ic555> ?p ?o .}")
	if err != nil {panic(err)}
	*/
	
}