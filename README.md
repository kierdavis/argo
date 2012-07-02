Argo
====

An RDF manipulation, parsing and serialisation library, written in Go.

[Documentation for package argo][argo-doc]

# Tutorials

## Using graphs

A *graph* is the primary unit for manipulating RDF. It is backed by a *store*, which is a basic
container for *triples*. Triples consist of 3 *terms*, which can be *resources* (URIs / IRIs),
*literals* or *blank nodes*.

The existing stores are:

*   **[argo.ListStore][liststore-doc]** - stores triples in memory in a slice of Triple objects.
    Better than an IndexStore at accumulating triples.
*   **[argo.IndexStore][indexstore-doc]** - stores triples in an indexed hierarchal structure.
    Better than a ListStore at querying.
*   **[mysqlstore.MySQLStore][mysqlstore-doc]** (incomplete): stores triples in a [MySQL][mysql]
    database.
*   **[redisstore.RedisStore][redisstore-doc]** (incomplete): stores triples in a [Redis][redis]
    database.

A graph can be created like using the [NewGraph][newgraph-doc] function:

    graph := argo.NewGraph(store)

For example, using a [ListStore][liststore-doc]:

    graph := argo.NewGraph(argo.NewListStore())

Triples can be added to graphs using the [AddTriple][graph-addtriple-doc] method:

    graph.AddTriple(subject, predicate, object)

In the above snippet, `subject`, `predicate` and `object` are all terms. Terms can be created in any
of the following ways:

    // A IRI reference - <http://example.com/>
    term = argo.NewResource("http://example.com/")
    
    // A named blank node - _:foobar
    term = argo.NewBlankNode("foobar")
    
    // An unnamed blank node; a unique identifier is assigned - _:b29
    term = argo.NewAnonNode()
    
    // A literal - "Hello, world!"
    term = argo.NewLiteral("Hello, world!")
    
    // A literal with a language - "Hello, world!"@en
    term = argo.NewLiteralWithLanguage("Hello, world!", "en")
    
    // A literal with a datatype - "Hello, world!"^^<http://vocab.example.org/greeting>
    term = argo.NewLiteralWithDatatype("Hello, world!",
        argo.NewResource("http://vocab.example.org/greeting"))

Triples can be removed in a similar way as they are added, using
[RemoveTriple][graph-removetriple-doc]:

    graph.RemoveTriple(subject, predicate, object)

## Prefixes and Namespaces

## Parsing and Serializing

Argo defines two types, [Parser][parser-doc] and [Serializer][serializer-doc] that can be used to
parse and serialize various RDF formats. The raw implementations of these types work on channels of
triples, for streaming large documents (see section [Streaming](#streaming). Therefore, the Graph type provides convience
wrappers ([Parse][graph-parse-doc] and [Serialize][graph-serialize-doc]) to ease parsing and
serializing with graphs.

The builtin parsers are currently:

*   **[argo.ParseNTriples][parsentriples-doc]** - NTriples (contributed by Ian Davis)
*   **[argo.ParseRDFXML][parserdfxml-doc]** - RDF/XML
*   **[rdfaparser][rdfaparser-doc]** - RDFA (does not directly implement Parser, see package doc for
    more)
*   **argo.ParseJSON** - RDF/JSON (planned)
*   **argo.ParseTurtle** - Turtle (planned)
*   **squirtle.ParseSquirtle** - [Squirtle][squirtle] (planned)

Likewise, the following serializers are implemented:

*   **[argo.SerializeNTriples][serializentriples-doc]** - NTriples
*   **[argo.SerializeRDFXML][serializerdfxml-doc]** - RDF/XML
*   **argo.SerializeJSON** - RDF/JSON (planned)
*   **argo.SerializeTurtle** - Turtle (planned)
*   **squirtle.SerializeSquirtle** - [Squirtle][squirtle] (planned)

A parser is used on a graph as follows:

    err = graph.Parse(argo.ParseRDFXML, reader)

where `reader` is an [io.Reader][io-reader-doc]. Serialization is almost identical:

    err = graph.Serialize(graph.SerializeRDFXML, writer)

However, even though you call the Serialize method on the graph, only the actual triples are passed
to the serializer; the prefix mapping is not. Therefore, the default RDF/XML serializer produces
rather ugly output. To improve this, a [SerializePrettyRDFXML][graph-serializeprettyrdfxml-doc]
method is provided on the graph, which takes a single argument (an [io.Writer][io-writer-doc]) and
returns an error object (or nil if no error occurred). This uses the graph's prefix mapping to
produce better-looking output:

    err = graph.SerializePrettyRDFXML(writer)

## Searching

Because Go implements channels natively, it is easy to use them as iterators. Therefore, most of the graph searching methods return a channel that yields successive results from a goroutine. The channel will be automatically closed by the goroutine when the end of the result set is reached, so it is possible to use a `for ... range` construction on it. These channels shall be referred to as *iterators* for the re

An iterator 

## Streaming

[argo-doc]:                         http://go.pkgdoc.org/github.com/kierdavis/argo
[graph-addtriple-doc]:              http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.AddTriple
[graph-parse-doc]:                  http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.Parse
[graph-removetriple-doc]:           http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.RemoveTriple
[graph-serialize-doc]:              http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.Serialize
[graph-serializeprettyrdfxml-doc]:  http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.SerializePrettyRDFXML
[indexstore-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo#IndexStore
[io-reader-doc]:                    http://golang.org/pkg/io/#Reader
[io-writer-doc]:                    http://golang.org/pkg/io/#Writer
[liststore-doc]:                    http://go.pkgdoc.org/github.com/kierdavis/argo#ListStore
[mysql]:                            http://www.mysql.com/
[mysqlstore-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo/mysqlstore#MySQLStore
[newgraph-doc]:                     http://go.pkgdoc.org/github.com/kierdavis/argo#NewGraph
[parsentriples-doc]:                http://go.pkgdoc.org/github.com/kierdavis/argo#ParseNTriples
[parser-doc]:                       http://go.pkgdoc.org/github.com/kierdavis/argo#Parser
[parserdfxml-doc]:                  http://go.pkgdoc.org/github.com/kierdavis/argo#ParseRDFXML
[redis]:                            http://redis.io/
[redisstore-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo/redisstore#RedisStore
[serializentriples-doc]:            http://go.pkgdoc.org/github.com/kierdavis/argo#SerializeNTriples
[serializer-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo#Serializer
[serializerdfxml-doc]:              http://go.pkgdoc.org/github.com/kierdavis/argo#SerializeRDFXML
[squirtle]:                         http://kierdavis.com/docs/squirtle
