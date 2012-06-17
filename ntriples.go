package argo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

// A ParseError is returned for parsing errors.
// The first line is 1.  The first column is 0.
type ParseError struct {
	Line   int   // Line where the error occurred
	Column int   // Column (rune index) where the error occurred
	Err    error // The actual error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Err)
}

// These are the errors that can be returned in ParseError.Error
var (
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrUnexpectedEOF       = errors.New("unexpected end of file")
	ErrTermCount           = errors.New("wrong number of terms in line")
	ErrUnterminatedIri     = errors.New("unterminated IRI, expecting '>'")
	ErrUnterminatedLiteral = errors.New("unterminated literal, expecting '\"'")
	ErrUnterminatedTriple  = errors.New("unterminated triple, expecting '.'")
)

type Reader struct {
	line   int
	column int
	r      *bufio.Reader
	buf    bytes.Buffer
}




// Constants for types of RdfTerm
const (
	RdfUnknown = iota
	RdfIri
	RdfBlank
	RdfLiteral
)

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

// error creates a new ParseError based on err.
func (r *Reader) error(err error) error {
	return &ParseError{
		Line:   r.line,
		Column: r.column,
		Err:    err,
	}
}

// Read reads the next triple
func (r *Reader) Read() (t *Triple, e error) {
	var s, p Term

	r.line++
	r.column = -1

	r1, err := r.skipWhitespace()
	if err != nil {
		return nil, err
	}

	for r1 == '#' {
		for {
			r1, err = r.readRune()
			if err != nil {
				return nil, err
			}
			if r1 == '\n' {
				break
			}
		}
		r1, err = r.skipWhitespace()
		if err != nil {
			return nil, err
		}

	}

	r.r.UnreadRune()

	termCount := 0
	for {
		haveTerm, term, err := r.parseTerm()
		if haveTerm {
			termCount++
			switch termCount {
			case 1:
				s = term
				err = r.expectWhitespace()
				if err != nil {
					return nil, err
				}
			case 2:
				p = term
				err = r.expectWhitespace()
				if err != nil {
					return nil, err
				}
			case 3:

				err = r.readEndTriple()
				if err != nil {
					return nil, err
				}

				return NewTriple(s, p, term), nil
			default:
				// TODO: error, too many terms
				return nil, r.error(ErrTermCount)
			}

		}
		if err != nil {
			return nil, err
		}
	}

	return nil, nil

}

// readRune reads one rune from r, folding \r\n to \n and keeping track
// of how far into the line we have read.  r.column will point to the start
// of this rune, not the end of this rune.
func (r *Reader) readRune() (rune, error) {
	r1, _, err := r.r.ReadRune()

	// Handle \r\n here.  We make the simplifying assumption that
	// anytime \r is followed by \n that it can be folded to \n.
	// We will not detect files which contain both \r\n and bare \n.
	if r1 == '\r' {
		r1, _, err = r.r.ReadRune()
		if err == nil {
			if r1 != '\n' {
				r.r.UnreadRune()
				r1 = '\r'
			}
		}
	}
	r.column++
	return r1, err
}

// unreadRune puts the last rune read from r back.
func (r *Reader) unreadRune() {
	r.r.UnreadRune()
	r.column--
}

func (r *Reader) parseTerm() (haveField bool, term Term, err error) {
	r.buf.Reset()

	r1, err := r.skipWhitespace()

	switch r1 {
	case '<':
		// Read an IRI
		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			if r1 == '>' {
				if r.buf.Len() == 0 {
					return false, term, r.error(ErrUnexpectedCharacter)
				}
				return true, NewResource(r.buf.String()), nil
			} else if r1 < 0x20 || r1 > 0x7E || r1 == ' ' || r1 == '<' || r1 == '"' {
				return false, term, r.error(ErrUnexpectedCharacter)
			}
			r.buf.WriteRune(r1)
		}
	case '_':
		// Read a blank node
		r1, err = r.readRune()
		if err != nil {
			if err == io.EOF {
				return false, term, r.error(ErrUnexpectedEOF)
			}
			return false, term, err
		}

		if r1 != ':' {
			return false, term, r.error(ErrUnexpectedCharacter)
		}

		r1, err = r.readRune()
		if err != nil {
			if err == io.EOF {
				return false, term, r.error(ErrUnexpectedEOF)
			}
			return false, term, err
		}
		if !((r1 >= 'a' && r1 <= 'z') || (r1 >= 'A' && r1 <= 'Z')) {
			return false, term, r.error(ErrUnexpectedCharacter)
		}
		r.buf.WriteRune(r1)

		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			if !((r1 >= 'a' && r1 <= 'z') || (r1 >= 'A' && r1 <= 'Z') || (r1 >= '0' && r1 <= '9')) {
				if r1 == '.' || unicode.IsSpace(r1) {
					r.unreadRune()
					return true, NewNode(r.buf.String()), nil
				}
				return false, term, r.error(ErrUnexpectedCharacter)
			}
			r.buf.WriteRune(r1)
		}

	case '"':
		// Read a literal
		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			switch r1 {
			case '"':
				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return false, term, r.error(ErrUnexpectedEOF)
					}
					return false, term, err
				}

				switch r1 {

				case '.', ' ', '\t':
					r.unreadRune()
					return true, NewLiteral(r.buf.String()), nil
				case '@':
					lexicalForm := r.buf.String()
					r.buf.Reset()

					for {
						r1, err = r.readRune()
						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}
						if r1 == '.' || r1 == ' ' || r1 == '\t' {
							if r.buf.Len() == 0 {
								return false, term, r.error(ErrUnexpectedCharacter)
							}
							return true, NewLiteralWithLanguage(lexicalForm,r.buf.String()) , nil
						}
						if r1 == '-' || (r1 >= 'a' && r1 <= 'z') || (r1 >= '0' && r1 <= '9') {
							r.buf.WriteRune(r1)
						} else {
							return false, term, r.error(ErrUnexpectedCharacter)
						}
					}
				case '^':
					lexicalForm := r.buf.String()
					r.buf.Reset()

					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return false, term, r.error(ErrUnexpectedEOF)
						}
						return false, term, err
					}
					if r1 != '^' {
						return false, term, r.error(ErrUnexpectedCharacter)
					}

					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return false, term, r.error(ErrUnexpectedEOF)
						}
						return false, term, err
					}
					if r1 != '<' {
						return false, term, r.error(ErrUnexpectedCharacter)
					}

					// Read an IRI
					for {
						r1, err = r.readRune()
						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}
						if r1 == '>' {
							if r.buf.Len() == 0 {
								return false, term, r.error(ErrUnexpectedCharacter)
							}
							return true, NewLiteralWithDatatype(lexicalForm,NewResource(r.buf.String())), nil
						} else if r1 < 0x20 || r1 > 0x7E || r1 == ' ' || r1 == '<' || r1 == '"' {
							return false, term, r.error(ErrUnexpectedCharacter)
						}
						r.buf.WriteRune(r1)
					}

				}
				return false, term, r.error(ErrUnexpectedCharacter)

			case '\\':
				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return false, term, r.error(ErrUnexpectedEOF)
					}
					return false, term, err
				}
				switch r1 {
				case '\\', '"':
				case 't':
					r1 = '\t'
				case 'r':
					r1 = '\r'
				case 'n':
					r1 = '\n'
				case 'u', 'U':

					codepoint := rune(0)

					for i := 3; i >= 0; i-- {
						r1, err = r.readRune()

						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}

						if r1 >= '0' && r1 <= '9' {
							codepoint += (1 << uint32(4*i)) * (r1 - '0')
						} else if r1 >= 'a' && r1 <= 'f' {
							codepoint += (1 << uint32(4*i)) * (r1 - 'a' + 10)
						} else if r1 >= 'A' && r1 <= 'F' {
							codepoint += (1 << uint32(4*i)) * (r1 - 'A' + 10)
						} else {
							return false, term, r.error(ErrUnexpectedCharacter)
						}

					}
					r1 = codepoint

				default:
					return false, term, r.error(ErrUnexpectedCharacter)
				}
			}
			r.buf.WriteRune(r1)
		}
	default:
		// TODO: raise error, unexpected character
		return false, term, r.error(ErrUnexpectedCharacter)

	}

	panic("unreachable")

}

func (r *Reader) readEndTriple() (err error) {
	r1, err := r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return r.error(ErrUnterminatedTriple)
		}
		return err
	}

	if r1 != '.' {
		return r.error(ErrUnexpectedCharacter)
	}

	r1, err = r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	if r1 != '\n' {
		return r.error(ErrUnexpectedCharacter)
	}

	return nil

}

func (r *Reader) skipWhitespace() (r1 rune, err error) {
	r1, err = r.readRune()
	if err != nil {
		return r1, err
	}

	for r1 == ' ' || r1 == '\t' {
		r1, err = r.readRune()
		if err != nil {
			return r1, err
		}
	}

	return r1, nil

}

func (r *Reader) expectWhitespace() (err error) {
	r1, err := r.readRune()
	if err != nil {
		if err == io.EOF {
			return r.error(ErrUnexpectedEOF)
		}
		return err
	}
	if r1 != ' ' && r1 != '\t' {
		return r.error(ErrUnexpectedCharacter)
	}

	return nil
}
