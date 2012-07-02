package argo

import ()

type toplevelIndex map[string]subjectIndex
type subjectIndex map[string][]Term

// An IndexStore stores triples as a hierarchal mapping structure, to speed up searching.
type IndexStore struct {
	index toplevelIndex
}

// Function NewIndexStore creates and returns a new Indexstore.
func NewIndexStore() (store *IndexStore) {
	return &IndexStore{
		index: make(toplevelIndex),
	}
}

// Method encodeKey converts a term object into a string.
func (store *IndexStore) encodeKey(term Term) (uri string) {
	res, isRes := term.(*Resource)
	if isRes {
		return res.URI
	}

	return "_:" + term.(*BlankNode).ID
}

// Method decodeKey converts a string as returned by encodeKey back into the term it represents.
func (store *IndexStore) decodeKey(uri string) (term Term) {
	if len(uri) >= 2 && uri[0] == '_' && uri[1] == ':' {
		return NewBlankNode(uri[2:])
	}

	return NewResource(uri)
}

// Method lookupSubject takes a subject term and returns the subjectIndex (a map of predicates to
// object lists) associated with it.
func (store *IndexStore) lookupSubject(subject Term) (subjIdx subjectIndex) {
	key := store.encodeKey(subject)
	subjIdx, ok := store.index[key]
	if !ok {
		subjIdx = make(subjectIndex)
		store.index[key] = subjIdx
	}

	return subjIdx
}

// Method lookupPredicate takes a subject index and a predicate term and returns the object list
// associated with it.
func (store *IndexStore) lookupPredicate(subjIdx subjectIndex, predicate Term) (objList []Term) {
	key := store.encodeKey(predicate)
	objList, ok := subjIdx[key]
	if !ok {
		objList = make([]Term, 0, 1)
		subjIdx[key] = objList
	}

	return objList
}

// Method storePredicate stores the given object list into the subject index, under the key given
// by the supplied predicate.
func (store *IndexStore) storePredicate(subjIdx subjectIndex, predicate Term, objList []Term) {
	key := store.encodeKey(predicate)
	subjIdx[key] = objList
}

// Method Add adds the given triple to the store.
func (store *IndexStore) Add(triple *Triple) {
	subjIdx := store.lookupSubject(triple.Subject)
	objList := store.lookupPredicate(subjIdx, triple.Predicate)

	store.storePredicate(subjIdx, triple.Predicate, append(objList, triple.Object))
}

// Method Remove removes the given triple from the store.
func (store *IndexStore) Remove(triple *Triple) {
	subjIdx := store.lookupSubject(triple.Subject)
	objList := store.lookupPredicate(subjIdx, triple.Predicate)

	for i, obj := range objList {
		if obj == triple.Object {
			store.storePredicate(subjIdx, triple.Predicate, append(objList[:i], objList[i+1:]...))
			break
		}
	}
}

// Method Clear empties the store.
func (store *IndexStore) Clear() {
	store.index = make(toplevelIndex)
}

// Method Num returns the number of triples in the store.
func (store *IndexStore) Num() (n int) {
	for _, subjIdx := range store.index {
		for _, objList := range subjIdx {
			n += len(objList)
		}
	}

	return n
}

// Method IterTriples returns a channel that yields successive triples in the graph.
func (store *IndexStore) IterTriples() (ch chan *Triple) {
	ch = make(chan *Triple)

	go func() {
		defer close(ch)

		for subjKey, subjIdx := range store.index {
			for predKey, objList := range subjIdx {
				for _, object := range objList {
					ch <- NewTriple(store.decodeKey(subjKey), store.decodeKey(predKey), object)
				}
			}
		}
	}()

	return ch
}

// Method Filter performs a basic filter; see the documentation of Store for information on the
// arguments.
func (store *IndexStore) Filter(subjSearch, predSearch, objSearch Term) (ch chan *Triple) {
	if subjSearch == nil && predSearch == nil && objSearch == nil {
		return store.IterTriples()
	}

	if subjSearch != nil {
		if predSearch != nil {
			if objSearch != nil {
				return store.filterSPO(subjSearch, predSearch, objSearch)
			}

			return store.filterSP(subjSearch, predSearch)
		}

		return store.filterS(subjSearch)
	}

	return store.filterDefault(subjSearch, predSearch, objSearch)
}

// Method filterSPO performs a filter when the subject, predicate and object are non-nil.
func (store *IndexStore) filterSPO(subjSearch, predSearch, objSearch Term) (ch chan *Triple) {
	/*
		ch = make(chan *Triple)

		subjIdx := store.lookupSubject(subjSearch)
		objList := store.lookupPredicate(subjIdx, predSearch)

		go func() {
			defer close(ch)

			for _, obj := range objList {
				if obj == objSearch {
					ch <- NewTriple(subjSearch, predSearch, objSearch)
				}
			}
		}()

		return ch
	*/

	ch = make(chan *Triple, 1)
	ch <- NewTriple(subjSearch, predSearch, objSearch)
	close(ch)
	return ch
}

// Method filterSP performs a filter when the subject and predicate are non-nil.
func (store *IndexStore) filterSP(subjSearch, predSearch Term) (ch chan *Triple) {
	ch = make(chan *Triple)

	subjIdx := store.lookupSubject(subjSearch)
	objList := store.lookupPredicate(subjIdx, predSearch)

	go func() {
		defer close(ch)

		for _, object := range objList {
			ch <- NewTriple(subjSearch, predSearch, object)
		}
	}()

	return ch
}

// Method filterS performs a filter when the subject is non-nil.
func (store *IndexStore) filterS(subjSearch Term) (ch chan *Triple) {
	ch = make(chan *Triple)

	subjIdx := store.lookupSubject(subjSearch)

	go func() {
		defer close(ch)

		for predKey, objList := range subjIdx {
			for _, object := range objList {
				ch <- NewTriple(subjSearch, store.decodeKey(predKey), object)
			}
		}
	}()

	return ch
}

// Method filterDefault performs a standard iteration filter.
func (store *IndexStore) filterDefault(subjSearch, predSearch, objSearch Term) (ch chan *Triple) {
	ch = make(chan *Triple)

	go func() {
		defer close(ch)

		for triple := range store.IterTriples() {
			if subjSearch != nil && subjSearch != triple.Subject {
				continue
			}

			if predSearch != nil && predSearch != triple.Predicate {
				continue
			}

			if objSearch != nil && objSearch != triple.Object {
				continue
			}

			ch <- triple
		}
	}()

	return ch
}
