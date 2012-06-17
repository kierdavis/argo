package argo

import (
	"fmt"
	"github.com/kierdavis/goutil"
	"io"
	"strings"
)

// Whitespace characters.
var Whitespace = " \t\r\n"

type NTriplesParseError struct {
	Message string
	Line    int
}

func (err *NTriplesParseError) Error() (msg string) {
	return fmt.Sprintf("Error when parsing line %d: %s", err.Line, err.Message)
}

func ParseNTriples(r io.Reader, tripleChan chan *Triple, errChan chan error) {
	defer close(tripleChan)
	defer close(errChan)

	var err error

	lineChan, lineErrChan := util.IterLines(r)
	lineno := 0

	for line := range lineChan {
		lineno++

		line = strings.Trim(line, Whitespace)
		if line == "" || line[0] == '#' {
			continue
		}

		line, subj, err := readNTriplesTerm(line, lineno)
		if err != nil {
			errChan <- err
			continue
		}

		line, pred, err := readNTriplesTerm(line, lineno)
		if err != nil {
			errChan <- err
			continue
		}

		line, obj, err := readNTriplesTerm(line, lineno)
		if err != nil {
			errChan <- err
			continue
		}

		var ctx Term

		if line[0] != '.' {
			line, ctx, err = readNTriplesTerm(line, lineno)
			if err != nil {
				errChan <- err
				continue
			}

			if line[0] != '.' {
				errChan <- &NTriplesParseError{"Unterminated line (lines must end with a period - '.')", lineno}
				continue
			}
		}

		tripleChan <- NewQuad(subj, pred, obj, ctx)
	}

	err = <-lineErrChan
	if err != nil {
		errChan <- err
	}
}

func SerialiseNTriples(w io.Writer, tripleChan chan *Triple, errChan chan error) {
	defer close(errChan)

	var err error

	for triple := range tripleChan {
		_, err = fmt.Fprintln(w, triple.String())
		if err != nil {
			errChan <- err
		}
	}
}

func readNTriplesTerm(line string, lineno int) (remainder string, term Term, err error) {
	line = strings.TrimLeft(line, Whitespace)

	// Resource
	if line[0] == '<' {
		end := strings.Index(line, ">")
		if end < 0 {
			return "", nil, &NTriplesParseError{"Unterminated resource URI (<...>); '>' character is required", lineno}
		}

		return line[end+1:], NewResource(line[1:end]), nil

		// Blank node
	} else if line[:2] == "_:" {
		end := strings.IndexAny(line, Whitespace)
		if end < 0 {
			return "", nil, &NTriplesParseError{"Unterminated blank node (_:...); delimiting whitespace is required", lineno}
		}

		return line[end+1:], NewNode(line[2:end]), nil

		// Literal
	} else if line[0] == '"' {
		end := strings.Index(line, "\"")
		if end < 0 {
			return "", nil, &NTriplesParseError{"Unterminated literal (\"...\"); '\"' character is required", lineno}
		}

		text := line[1:end]
		line = line[end+1:]

		if line[0] == '@' {
			end = strings.IndexAny(line, Whitespace)
			if end < 0 {
				return "", nil, &NTriplesParseError{"Unterminated language identifier (@...); delimiting whitespace is required", lineno}
			}

			return line[end+1:], NewLiteralWithLanguage(text, line[1:end]), nil

		} else if strings.HasPrefix(line, "^^") {
			remainder, datatype, err := readNTriplesTerm(line[2:], lineno)
			if err != nil {
				return "", nil, err
			}

			return remainder, NewLiteralWithDatatype(text, datatype), nil
		}

		return line, NewLiteral(text), nil
	}

	return "", nil, &NTriplesParseError{"Invalid term syntax", lineno}
}
