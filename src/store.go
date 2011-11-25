package rdflib

import (
	"container/vector"
)

type Store interface {
	Add(triple *Triple)
	Remove(triple *Triple)
	Clear()
	ResetIter()
	Next() (triple *Triple, ok bool)
}


type MemoryStore struct {
	triples []*Triple
	iter int
}

func CreateMemoryStore() (store *MemoryStore) {
	store = &MemoryStore{iter: 0, triples: []*Triple{}}
	return store
}

func (store *MemoryStore) Add(triple *Triple) {
	store.triples = append(store.triples, triple)
}

func (store *MemoryStore) Remove(triple *Triple) {
	for i, t := range store.triples {
		if t.EqualTo(triple) {
			(*vector.Vector(store.triples)).Delete(i)
		}
	}
}

func (store *MemoryStore) Clear() {
	store.triples = []*Triple{}
}

func (store *MemoryStore) ResetIter() {
	store.iter = 0
}

func (store *MemoryStore) Next() (triple *Triple, ok bool) {
	if store.iter >= len(store.triples) {
		return nil, false
	}

	triple = store.triples[store.iter]
	store.iter++
	return triple, true
}