package argo

import ()

type toplevelIndex map[string]subjectIndex
type subjectIndex map[string][]Term

type IndexStore struct {
	index toplevelIndex
}

func NewIndexStore() (store *IndexStore) {
	return &IndexStore{
		index: make(toplevelIndex),
	}
}

func (store *IndexStore) encodeKey(term Term) (uri string) {
	res, isRes := term.(*Resource)
	if isRes {
		return res.URI
	}

	return "_:" + term.(*BlankNode).ID
}

func (store *IndexStore) decodeKey(uri string) (term Term) {
	if len(uri) >= 2 && uri[0] == '_' && uri[1] == ':' {
		return NewBlankNode(uri[2:])
	}

	return NewResource(uri)
}

func (store *IndexStore) lookupSubject(subject Term) (subjIdx subjectIndex) {
	key := store.encodeKey(subject)
	subjIdx, ok := store.index[key]
	if !ok {
		subjIdx = make(subjectIndex)
		store.index[key] = subjIdx
	}

	return subjIdx
}

func (store *IndexStore) lookupPredicate(subjIdx subjectIndex, predicate Term) (objList []Term) {
	key := store.encodeKey(predicate)
	objList, ok := subjIdx[key]
	if !ok {
		objList = make([]Term, 0, 1)
		subjIdx[key] = objList
	}

	return objList
}

func (store *IndexStore) storePredicate(subjIdx subjectIndex, predicate Term, objList []Term) {
	key := store.encodeKey(predicate)
	subjIdx[key] = objList
}

func (store *IndexStore) SupportsIndexes() (result bool) {
	return false
}

func (store *IndexStore) Add(triple *Triple) (index int) {
	subjIdx := store.lookupSubject(triple.Subject)
	objList := store.lookupPredicate(subjIdx, triple.Predicate)

	store.storePredicate(subjIdx, triple.Predicate, append(objList, triple.Object))

	return 0
}

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

func (store *IndexStore) RemoveIndex(index int) {
	panic("not implemented!")
}

func (store *IndexStore) Clear() {
	store.index = make(toplevelIndex)
}

func (store *IndexStore) Num() (n int) {
	for _, subjIdx := range store.index {
		for _, objList := range subjIdx {
			n += len(objList)
		}
	}

	return n
}

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

func (store *IndexStore) filterSPO(subjSearch, predSearch, objSearch Term) (ch chan *Triple) {
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
}

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
