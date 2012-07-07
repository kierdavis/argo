package argo

import (
	"fmt"
	"io"
)

func SerializeJSON(w io.Writer, tripleChan chan *Triple, errChan chan error, prefixes map[string]string) {
	defer close(errChan)

	// Offload indexing to IndexStore

	store := NewIndexStore()
	for triple := range tripleChan {
		store.Add(triple)
	}

	_, err := fmt.Fprint(w, "{")
	if err != nil {
		errChan <- err
		return
	}

	restSubject := false

	for subject, predicates := range store.index {
		if restSubject {
			_, err = fmt.Fprint(w, ",")
			if err != nil {
				errChan <- err
				return
			}

		} else {
			restSubject = true
		}

		_, err = fmt.Fprintf(w, "'%s':{", subject)
		if err != nil {
			errChan <- err
			return
		}

		restPredicate := false

		for predicate, objects := range predicates {
			if restPredicate {
				_, err = fmt.Fprint(w, ",")
				if err != nil {
					errChan <- err
					return
				}

			} else {
				restPredicate = true
			}

			_, err = fmt.Fprintf(w, "'%s':[", predicate)
			if err != nil {
				errChan <- err
				return
			}

			restObject := false

			for _, object := range objects {
				if restObject {
					_, err = fmt.Fprint(w, ",")
					if err != nil {
						errChan <- err
						return
					}

				} else {
					restObject = true
				}

				switch o := object.(type) {
				case *Resource:
					_, err = fmt.Fprintf(w, "{'type':'uri','value':'%s'}", o.URI)

				case *BlankNode:
					_, err = fmt.Fprintf(w, "{'type':'bnode','value':'_:%s'}", o.ID)

				case *Literal:
					if o.Language != "" {
						_, err = fmt.Fprintf(w, "{'type':'literal','value':'%s','lang':'%s'}", o.Value, o.Language)
					} else if o.Datatype != nil {
						_, err = fmt.Fprintf(w, "{'type':'literal','value':'%s','datatype':'%s'}", o.Value, o.Datatype.(*Resource).URI)
					} else {
						_, err = fmt.Fprintf(w, "{'type':'literal','value':'%s'}", o.Value)
					}
				}

				if err != nil {
					errChan <- err
					return
				}
			}

			_, err = fmt.Fprint(w, "]")
			if err != nil {
				errChan <- err
				return
			}
		}

		_, err = fmt.Fprint(w, "}")
		if err != nil {
			errChan <- err
			return
		}
	}

	_, err = fmt.Fprint(w, "}")
	if err != nil {
		errChan <- err
		return
	}
}
