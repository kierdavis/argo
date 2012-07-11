package argo

import (
	"fmt"
	"io"
)

func SerializeSquirtle(w io.Writer, tripleChan chan *Triple, errChan chan error, prefixes map[string]string) {
	defer close(errChan)

	var err error

	triplesBySubject := make(map[string][]*Triple)

	encodeTerm := func(iterm Term) (s string) {
		switch term := iterm.(type) {
		case *Resource:
			base, local := SplitPrefix(term.URI)
			prefix, ok := prefixes[base]
			if ok {
				return fmt.Sprintf("%s:%s", prefix, local)
			} else {
				return fmt.Sprintf("<%s>", term.URI)
			}

		case *Literal:
			return term.String()

		case *BlankNode:
			return term.String()
		}

		return ""
	}

	var describe func(string, []*Triple, string) bool

	describe = func(subject string, triples []*Triple, ind string) (r bool) {
		_, err = fmt.Fprintf(w, "%s {\n", subject)
		if err != nil {
			errChan <- err
			return false
		}

		for _, triple := range triples {
			p := encodeTerm(triple.Predicate)
			o := encodeTerm(triple.Object)

			_, err = fmt.Fprintf(w, "%s  %s ", ind, p)
			if err != nil {
				errChan <- err
				return false
			}

			objectTriples, ok := triplesBySubject[o]
			if ok {
				delete(triplesBySubject, o)
				if !describe(o, objectTriples, ind+"  ") {
					return false
				}

			} else {
				_, err = fmt.Fprintln(w, o)
				if err != nil {
					errChan <- err
					return false
				}
			}
		}

		_, err = fmt.Fprintf(w, "%s}\n", ind)
		if err != nil {
			errChan <- err
			return false
		}

		return true
	}

	for triple := range tripleChan {
		s := encodeTerm(triple.Subject)
		triplesBySubject[s] = append(triplesBySubject[s], triple)
	}

	for base, prefix := range prefixes {
		_, err = fmt.Fprintf(w, "name <%s> as %s\n", base, prefix)
		if err != nil {
			errChan <- err
			return
		}
	}

	_, err = fmt.Fprint(w, "\n")
	if err != nil {
		errChan <- err
		return
	}

	for subject, triples := range triplesBySubject {
		if !describe(subject, triples, "") {
			return
		}
	}
}
