import sys
import rdflib

FOAF = rdflib.Namespace("http://xmlns.com/foaf/0.1/")
EXAMPLE = rdflib.Namespace("http://example.org/foo/")

if __name__ == "__main__":
	g = rdflib.ConjunctiveGraph()
	g.bind("foaf", FOAF)
	g.bind("ex", EXAMPLE)

	subj = rdflib.URIRef("http://kierdavis.tklapp.com/me")

	g.add((subj, FOAF["name"], rdflib.Literal("Kier", datatype=EXAMPLE["name"])))
	g.add((subj, FOAF["made"], rdflib.URIRef("http://kierdavis.tklapp.com/")))

	g.serialize(sys.stdout, format="xml")
