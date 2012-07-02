Argo
====

An RDF manipulation, parsing and serialisation library, written in Go.

[Documentation for package argo][argo-doc]

# Tutorials

## Using graphs

A *graph* is the primary unit for manipulating RDF. It is backed by a *store*, which is a basic container for *triples*. Triples consist of 3 *terms*, which can be *resources* (URIs / IRIs), *literals* or *blank nodes*.

The existing stores are:

*   **[argo.ListStore][liststore-doc]** - stores triples in memory in a slice of Triple objects. Better than an IndexStore at accumulating triples.
*   **[argo.IndexStore][indexstore-doc]** - stores triples in an indexed hierarchal structure. Better than a ListStore at querying.
*   **[mysqlstore.MySQLStore][mysqlstore-doc]** (incomplete): stores triples in a [MySQL][mysql] database.
*   **[redisstore.RedisStore][redisstore-doc]** (incomplete): stores triples in a [Redis][redis] database.

A graph can be created like using the [NewGraph][newgraph-doc] function:

    graph := argo.NewGraph(store)

For example, using a [ListStore][liststore-doc]:

    graph := argo.NewGraph(argo.NewListStore())

Triples can be added to graphs using the [AddTriple][graph-addtriple-doc] method:

    graph.AddTriple(subject, predicate, object)

In the above snippet, `subject`, `predicate` and `object` are all terms. Terms can be created in any of the following ways:

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

Argo defines two types, [Parser][parser-doc] and [Serializer][serializer-doc] that can be used to parse and serialize various RDF formats. The raw implementations of these types work on channels of triples, for streaming large documents (see section [Streaming](#streaming). Therefore, the Graph type provides convience wrappers ([Parse][graph-parse-doc] and [Serialize][graph-serialize-doc]) to ease parsing and serializing with graphs.

The builtin parsers are currently:

*   **[argo.ParseNTriples][parsentriples-doc]** - NTriples (contributed by Ian Davis)
*   **[argo.ParseRDFXML][parserdfxml-doc]** - RDF/XML
*   **[rdfaparser][rdfaparser-doc]** - RDFA (does not directly implement Parser, see package doc for more)
*   **[squirtle.ParseSquirtle][parsesquirtle-doc]** - [Squirtle][squirtle]
*   **argo.ParseJSON** - RDF/JSON (planned)
*   **argo.ParseTurtle** - Turtle (planned)

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

## Searching

Because Go implements channels natively, it is easy to use them as iterators. Therefore, most of the graph searching methods return a channel that yields successive results from a goroutine. The channel will be automatically closed by the goroutine when the end of the result set is reached, so it is possible to use a `for ... range` construction on it. The reason for using these channels instead of simply returning a slice of triples is that very large datasets may be returned from large graphs, and for persistent stores that store the data on disk instead of in memory (such as MySQL and Redis) it is more memory efficient to process the data one triple at a time. These channels shall be referred to as *iterators* for the remainder of this document.

An iterator over all triples in a graph can be returned by the [IterTriples][graph-itertriples-doc] method:

    for triple := range graph.IterTriples() {
        // do something with triple
    }

A basic filtering method is implemented too. If the subject argument to the [Filter][graph-filter-doc] method is not nil, then only triples with that subject will be returned. The same goes for the predicate and object arguments. The following example:

    me := graph.NewResource("http://kierdavis.com/data/me")

    for triple := range graph.Filter(me, argo.FOAF.Get("knows"), nil) {
        // do something with triple.Object
    }

gives all triples that have a subject of `<http://kierdavis.com/data/me>` and a predicate of `foaf:knows`. Any object is allowed, because the 3rd argument (the object argument) is nil.

There are other methods based off this Filter method. [GetAll][graph-getall-doc] returns all objects with the given subject and predicate, as a slice. [Get][graph-get-doc] returns the first object with the given subject and predicate, or nil. [MustGet][graph-mustget-doc] either returns the first object with the given subject and predicate, or panics if one was not found.

## Streaming

## Containers and Lists

## Other Methods



[argo-doc]:                         http://go.pkgdoc.org/github.com/kierdavis/argo
[graph-addtriple-doc]:              http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.AddTriple
[graph-filter-doc]:                 http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.Filter
[graph-get-doc]:                    http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.Get
[graph-getall-doc]:                 http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.GetAll
[graph-itertriples-doc]:            http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.IterTriples
[graph-mustget-doc]:                http://go.pkgdoc.org/github.com/kierdavis/argo#Graph.MustGet
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
[parsesquirtle-doc]:                http://go.pkgdoc.org/github.com/kierdavis/argo/squirtle#ParseSquirtle
[redis]:                            http://redis.io/
[redisstore-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo/redisstore#RedisStore
[serializentriples-doc]:            http://go.pkgdoc.org/github.com/kierdavis/argo#SerializeNTriples
[serializer-doc]:                   http://go.pkgdoc.org/github.com/kierdavis/argo#Serializer
[serializerdfxml-doc]:              http://go.pkgdoc.org/github.com/kierdavis/argo#SerializeRDFXML
[squirtle]:                         http://kierdavis.com/docs/squirtle
