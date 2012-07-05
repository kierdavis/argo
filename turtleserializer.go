package argo

import (
	"fmt"
	"io"
)

func SerializeTurtle(w io.Writer, tripleChan chan *Triple, errChan chan error, prefixes map[string]string) {
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

	for triple := range tripleChan {
		s := encodeTerm(triple.Subject)
		triplesBySubject[s] = append(triplesBySubject[s], triple)
	}

	for base, prefix := range prefixes {
		_, err = fmt.Fprintf(w, "@prefix %s: <%s> .\n", prefix, base)
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
		_, err = fmt.Fprintf(w, "%s\n", subject)
		if err != nil {
			errChan <- err
			return
		}

		for _, triple := range triples {
			p := encodeTerm(triple.Predicate)
			o := encodeTerm(triple.Object)

			_, err = fmt.Fprintf(w, "  %s %s ;\n", p, o)
			if err != nil {
				errChan <- err
				return
			}
		}

		_, err = fmt.Fprintf(w, "  .\n\n")
		if err != nil {
			errChan <- err
			return
		}
	}
}
