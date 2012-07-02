package argo

import ()

// A ListStore is a Store that stores triples in a slice stored in memory.
type ListStore struct {
	triples []*Triple
}

// Function NewListStore create and returns a new empty ListStore.
func NewListStore() (store *ListStore) {
	return &ListStore{
		triples: make([]*Triple, 0),
	}
}

// Method Add adds the given triple to the store and returns its index.
func (store *ListStore) Add(triple *Triple) {
	store.triples = append(store.triples, triple)
}

// Method Remove removes the given triple from the store.
func (store *ListStore) Remove(triple *Triple) {
	for i, t := range store.triples {
		if t == triple {
			store.triples = append(store.triples[:i], store.triples[i+1:]...)
			return
		}
	}
}

// Method Clear removes all triples from the store.
func (store *ListStore) Clear() {
	store.triples = store.triples[:0]
}

// Method Num returns the number of triples in the store.
func (store *ListStore) Num() (n int) {
	return len(store.triples)
}

// Method IterTriples returns a channel that will yield the triples of the store. The channel will
// be closed when iteration is completed.
func (store *ListStore) IterTriples() (ch chan *Triple) {
	ch = make(chan *Triple)

	go func() {
		for _, triple := range store.triples {
			ch <- triple
		}

		close(ch)
	}()

	return ch
}

// Method Filter returns a channel that will yield all matching triples of the graph. A nil value
// passed means that the check for this term is skipped; else the triples returned must have the
// same terms as the corresponding arguments.
func (store *ListStore) Filter(subject Term, predicate Term, object Term) (ch chan *Triple) {
	ch = make(chan *Triple)

	go func() {
		for _, triple := range store.triples {
			if subject != nil && subject != triple.Subject {
				continue
			}

			if predicate != nil && predicate != triple.Predicate {
				continue
			}

			if object != nil && object != triple.Object {
				continue
			}

			ch <- triple
		}

		close(ch)
	}()

	return ch
}
