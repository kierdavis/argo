package argo

import ()

type ListStore struct {
	triples []*Triple
}

func NewListStore(capacity int) (store *ListStore) {
	return &ListStore{
		triples: make([]*Triple, 0, capacity),
	}
}

func (store *ListStore) Add(triple *Triple) (index int) {
	index = len(store.triples)
	store.triples = append(store.triples, triple)
	return index
}

func (store *ListStore) Remove(triple *Triple) {
	for i, t := range store.triples {
		if t == triple {
			store.RemoveIndex(i)
			return
		}
	}
}

func (store *ListStore) RemoveIndex(index int) {
	store.triples = append(store.triples[:index], store.triples[index+1:]...)
}

func (store *ListStore) Clear() {
	store.triples = store.triples[:0]
}

func (store *ListStore) Num() (n int) {
	return len(store.triples)
}

func (store *ListStore) IterTriples() (ch chan *Triple) {
	ch1 := make(chan *Triple)

	go func() {
		for _, triple := range store.triples {
			ch1 <- triple
		}

		close(ch1)
	}()

	return ch1
}
