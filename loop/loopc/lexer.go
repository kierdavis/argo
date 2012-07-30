package main

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const EOF rune = -1

type tokenType int

const (
	// Control tokens
	tokError tokenType = iota + 256
	tokEOF

	// Normal tokens
	tokIdentifier
	tokNumber
	tokURIRef
	tokQuotedString
	tokDTSymbol

	// Keyword tokens
	tokAs
	tokBoolean
	tokData
	tokFalse
	tokFloat
	tokFunc
	tokInteger
	tokName
	tokOf
	tokResource
	tokString
	tokTrue
)

func (tokType tokenType) String() (str string) {
	switch tokType {
	case tokError:
		return "ERROR"
	case tokEOF:
		return "EOF"
	case tokIdentifier:
		return "IDENTIFIER"
	case tokNumber:
		return "NUMBER"
	case tokURIRef:
		return "URIREF"
	case tokQuotedString:
		return "QUOTEDSTRING"
	case tokDTSymbol:
		return "DTSYMBOL"
	case tokAs:
		return "AS"
	case tokBoolean:
		return "BOOLEAN"
	case tokData:
		return "DATA"
	case tokFalse:
		return "FALSE"
	case tokFloat:
		return "FLOAT"
	case tokFunc:
		return "FUNC"
	case tokInteger:
		return "INTEGER"
	case tokName:
		return "NAME"
	case tokOf:
		return "OF"
	case tokResource:
		return "RESOURCE"
	case tokString:
		return "STRING"
	case tokTrue:
		return "TRUE"
	}

	return string(tokType)
}

type token struct {
	typ    tokenType
	val    string
	lineno int
	column int
}

func (tok token) String() (str string) {
	if tok.typ == tokError {
		return tok.val
	}

	if tok.typ == tokEOF {
		return "EOF"
	}

	if len(tok.val) > 20 {
		return fmt.Sprintf("%s(%.20s...)", tok.typ, tok.val)
	}

	return fmt.Sprintf("%s(%s)", tok.typ, tok.val)
}

type stateFunc func(*lexer) stateFunc
type validatorFunc func(rune) bool

type lexer struct {
	name          string
	input         string
	start         int
	pos           int
	width         int
	lineno        int
	lastLineno    int
	column        int
	lastColumn    int
	lastWasReturn bool
	state         stateFunc
	tokens        chan token
}

func newLexer(name string, input string) (l *lexer) {
	return &lexer{
		name:   name,
		input:  input,
		lineno: 1,
		column: 1,
		state:  lexTop,
		tokens: make(chan token, 2),
	}
}

func (l *lexer) lex() (tok token) {
loop:
	for {
		select {
		case tok = <-l.tokens:
			return tok

		default:
			if l.state == nil {
				break loop
			}

			l.state = l.state(l)
		}
	}

	return token{tokEOF, "", l.lineno, l.column}
}

func (l *lexer) emit(tokType tokenType) {
	s := l.input[l.start:l.pos]
	l.tokens <- token{tokType, s, l.lineno, l.column - utf8.RuneCountInString(s)}
	l.start = l.pos
}

func (l *lexer) peek() (c rune) {
	c = l.next()
	l.backup()
	return c
}

func (l *lexer) next() (c rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return EOF
	}

	c, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width

	l.lastLineno = l.lineno
	l.lastColumn = l.column

	switch c {
	case '\r':
		l.lineno++
		l.column = 1
		l.lastWasReturn = true

	case '\n':
		if !l.lastWasReturn {
			l.lineno++
			l.column = 1
		}

		l.lastWasReturn = false

	default:
		l.column++
		l.lastWasReturn = false
	}

	return c
}

func (l *lexer) backup() {
	l.pos -= l.width
	l.lineno = l.lastLineno
	l.column = l.lastColumn
}

func (l *lexer) accept(valid string) (ok bool) {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}

	l.backup()
	return false
}

func (l *lexer) acceptFunc(validator validatorFunc) (ok bool) {
	if validator(l.next()) {
		return true
	}

	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	c := l.next()
	for strings.IndexRune(valid, c) >= 0 {
		c = l.next()
	}

	if c != EOF {
		l.backup()
	}
}

func (l *lexer) acceptRunFunc(validator validatorFunc) {
	c := l.next()
	for validator(c) {
		c = l.next()
	}

	if c != EOF {
		l.backup()
	}
}

func (l *lexer) error(s string, args ...interface{}) (nextState stateFunc) {
	col := l.column - utf8.RuneCountInString(l.input[l.start:l.pos])

	s = fmt.Sprintf(s, args...)
	s = fmt.Sprintf("[line %d col %d] %s", l.lineno, col, s)

	l.tokens <- token{tokError, s, l.lineno, col}

	// Stop the state machine
	//return nil 

	// Skip the token and go back to lexTop
	l.start = l.pos
	return lexTop
}

func lexTop(l *lexer) (nextState stateFunc) {
	l.acceptRunFunc(unicode.IsSpace)

	c := l.next()

	// Skip all input before this character
	l.start = l.pos - l.width

	switch {
	case unicode.IsLetter(c):
		return lexIdentifier

	case c >= '0' && c <= '9', c == '+', c == '-':
		return lexNumber

	default:
		switch c {
		case EOF:
			return nil

		case '<':
			return lexURIRef

		case '"':
			return lexString

		case '^':
			if l.accept("^") {
				l.emit(tokDTSymbol)
				return lexTop

			} else {
				return l.error("Invalid character %q", c)

				//l.emit('^')
				//return lexTop
			}

		case '(', ')', '{', '}', '/', ':', ',':
			l.emit(tokenType(c))
			return lexTop

		default:
			return l.error("Invalid character %q", c)
		}
	}

	panic("unreachable!")
}

func lexIdentifier(l *lexer) (nextState stateFunc) {
	l.acceptRunFunc(func(c rune) bool {
		return c == '_' || c == '-' || unicode.IsLetter(c) || unicode.IsDigit(c)
	})

	word := l.input[l.start:l.pos]

	switch strings.ToLower(word) {
	case "as":
		l.emit(tokAs)
	case "boolean", "bool":
		l.emit(tokBoolean)
	case "data":
		l.emit(tokData)
	case "false":
		l.emit(tokFalse)
	case "float":
		l.emit(tokFloat)
	case "func":
		l.emit(tokFunc)
	case "integer", "int":
		l.emit(tokInteger)
	case "name":
		l.emit(tokName)
	case "of":
		l.emit(tokOf)
	case "resource", "res":
		l.emit(tokResource)
	case "string", "str":
		l.emit(tokString)
	case "true":
		l.emit(tokTrue)
	default:
		l.emit(tokIdentifier)
	}

	return lexTop
}

func lexNumber(l *lexer) (nextState stateFunc) {
	l.accept("+-")

	digits := "0123456789"

	// Is it hex?
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}

	// Accept digits
	l.acceptRun(digits)

	// Accept decimal point and more digits
	if l.accept(".") {
		l.acceptRun(digits)
	}

	// Accept exponent
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}

	if unicode.IsLetter(l.peek()) {
		l.next()
		return l.error("Bad number syntax: %q", l.input[l.start:l.pos])
	}

	l.emit(tokNumber)

	return lexTop
}

func lexURIRef(l *lexer) (nextState stateFunc) {
	// Current token value is the '<', drop it
	l.start = l.pos

	// Read until the '>'
	l.acceptRunFunc(func(c rune) bool { return c != '>' })
	l.emit(tokURIRef)

	// Skip the '>'
	l.next()
	l.start = l.pos

	return lexTop
}

func lexString(l *lexer) (nextState stateFunc) {
	// Current token value is the '"', drop it
	l.start = l.pos

	// Read until the '"'
	l.acceptRunFunc(func(c rune) bool { return c != '"' })
	l.emit(tokQuotedString)

	// Skip the '"'
	l.next()
	l.start = l.pos

	return lexTop
}
