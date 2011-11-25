all: build/ build/test
	@echo "Run build/test to show an example"

GOC = gccgo
GOC_FLAGS = -g -Wall -O2 -Ibuild

SRCS = src/term.go src/triple.go src/store.go src/graph.go src/namespace.go

build/:
	mkdir build/

build/test: build/rdflib.o
	$(GOC) $(GOC_FLAGS) -o build/test test.go build/rdflib.o

build/rdflib.o: $(SRCS)
	$(GOC) $(GOC_FLAGS) -o build/rdflib.o -c $(SRCS)

clean:
	rm -rf build
