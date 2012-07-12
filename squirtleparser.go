//line src/github.com/kierdavis/argo/squirtleparser.y:3

/*
   Copyright (c) 2012 Kier Davis

   Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
   associated documentation files (the "Software"), to deal in the Software without restriction,
   including without limitation the rights to use, copy, modify, merge, publish, distribute,
   sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
   furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in all copies or substantial
   portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
   NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES
   OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package argo

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type sqtlVar struct {
	Name string
}

func (v *sqtlVar) String() (str string) {
	return "?" + v.Name
}

func (v *sqtlVar) Equal(other Term) (result bool) {
	if spec, ok := other.(*sqtlVar); ok {
		return spec.Name == v.Name
	}

	return false
}

type sqtlStackEntry struct {
	NextItem int
	Subject  Term
	Template []*Triple
}

type sqtlTemplate struct {
	ArgNames []string
	Triples  []*Triple
}

var sqtlParserMutex sync.Mutex

var sqtlTripleChan chan *Triple
var sqtlErrChan chan error
var sqtlPrefixMap map[string]string

var sqtlNames map[string]string
var sqtlTemplates map[string]*sqtlTemplate
var sqtlStack []sqtlStackEntry

//line src/github.com/kierdavis/argo/squirtleparser.y:74
type sqtlSymType struct {
	yys int
	s   string
	sL  []string
	t   Term
	tL  []Term
}

const A_KWD = 57346
const AS = 57347
const BNODE = 57348
const DECIMAL = 57349
const DOUBLE = 57350
const DT = 57351
const EOF = 57352
const FALSE = 57353
const IDENTIFIER = 57354
const INCLUDE = 57355
const INTEGER = 57356
const IRIREF = 57357
const IS = 57358
const NAME = 57359
const NEW = 57360
const STRING = 57361
const TEMPLATE = 57362
const TRUE = 57363
const VAR = 57364

var sqtlToknames = []string{
	"A_KWD",
	"AS",
	"BNODE",
	"DECIMAL",
	"DOUBLE",
	"DT",
	"EOF",
	"FALSE",
	"IDENTIFIER",
	"INCLUDE",
	"INTEGER",
	"IRIREF",
	"IS",
	"NAME",
	"NEW",
	"STRING",
	"TEMPLATE",
	"TRUE",
	"VAR",
}
var sqtlStatenames = []string{}

const sqtlEofCode = 1
const sqtlErrCode = 2
const sqtlMaxDepth = 200

//line src/github.com/kierdavis/argo/squirtleparser.y:316

func getName(name string) (uri string) {
	uri, ok := sqtlNames[name]
	if ok {
		return uri
	}

	uri, err := LookupPrefix(name)
	if err == nil {
		sqtlNames[name] = uri
		sqtlPrefixMap[uri] = name
		return uri
	}

	return ""
}

func addHash(s string) (r string) {
	if s == "" {
		return "#"
	}

	last := s[len(s)-1]
	if last != '#' && last != '/' {
		return s + "#"
	}

	return s
}

func stripSlash(s string) (r string) {
	if s == "" {
		return ""
	}

	last := s[len(s)-1]
	if last == '#' || last == '/' {
		return s[:len(s)-1]
	}

	return s
}

type lexer struct {
	input           *bufio.Reader
	currentToken    []rune
	currentTokenLen int
	lastTokenLen    int
	lastColumn      int
	lineno          int
	column          int
}

func newLexer(input io.Reader) (ll *lexer) {
	ll = &lexer{
		input:  bufio.NewReader(input),
		lineno: 1,
		column: 1,
	}

	return ll
}

func (ll *lexer) Error(s string) {
	sqtlErrChan <- fmt.Errorf("Syntax error: %s (at line %d col %d)", s, ll.lineno, ll.column)
	//panic("foobar")
}

func (ll *lexer) Lex(lval *sqtlSymType) (t int) {
	ll.AcceptRun(unicode.IsSpace)
	ll.Discard()

	r := ll.Next()

	switch {
	case r == '_':
		if ll.Accept(':') {
			ll.Discard()
			return BNODE
		}

		fallthrough

	case unicode.IsLetter(r):
		ll.AcceptRun(func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' })
		lval.s = ll.GetToken()

		switch strings.ToLower(lval.s) {
		case "a":
			return A_KWD

		case "as":
			return AS

		case "false":
			return FALSE

		case "include":
			return INCLUDE

		case "inf":
			lval.s = "INF"
			return DOUBLE

		case "is":
			return IS

		case "name":
			return NAME

		case "nan":
			lval.s = "NaN"
			return DOUBLE

		case "new":
			return NEW

		case "template":
			return TEMPLATE

		case "true":
			return TRUE
		}

		return IDENTIFIER

	case unicode.IsDigit(r) || r == '-' || r == '+':
		ll.Back()
		ll.AcceptOneOf('+', '-')

		t = INTEGER

		digitFunc := func(r rune) bool { return '0' <= r && r <= '9' }

		ll.AcceptRun(digitFunc)
		if ll.Accept('.') {
			ll.AcceptRun(digitFunc)
			t = DECIMAL
		}

		if ll.AcceptOneOf('e', 'E') {
			ll.AcceptOneOf('+', '-')
			ll.AcceptRun(digitFunc)
			t = DOUBLE
		}

		p := ll.Peek()
		if unicode.IsLetter(p) || ('0' <= p && p <= '9') {
			ll.Next()
			return ll.Lex(lval)
		}

		lval.s = ll.GetToken()
		return t

	case r == '#':
		for ll.Next() != '\n' {
		}

		return ll.Lex(lval)

	case r == '?', r == '$':
		ll.Discard()
		ll.AcceptRun(func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' })
		lval.s = ll.GetToken()

		return VAR

	case r == '<':
		ll.Discard()

		ll.AcceptRun(func(r rune) bool { return r != '>' })
		lval.s = ll.GetToken()

		ll.Next()
		ll.Discard()

		return IRIREF

	case r == '"':
		ll.Discard()

		ll.AcceptRun(func(r rune) bool { return r != '"' })
		lval.s = ll.GetToken()

		ll.Next()
		ll.Discard()

		return STRING

	case r == '^':
		if ll.Accept('^') {
			ll.Discard()
			return DT
		}

		ll.Discard()
		return '^'
	}

	ll.Discard()
	return int(r)
}

func (ll *lexer) Next() (r rune) {
	r, n, err := ll.input.ReadRune()
	if err != nil {
		if err == io.EOF {
			return EOF
		}

		ll.Error(err.Error())
	}

	ll.currentToken = append(ll.currentToken, r)
	ll.currentTokenLen += n
	ll.lastTokenLen = n
	ll.lastColumn = ll.column

	if r == '\n' {
		ll.lineno++
		ll.column = 1
	} else {
		ll.column++
	}

	return r
}

func (ll *lexer) Back() {
	err := ll.input.UnreadRune()
	if err == nil {
		if ll.currentToken[len(ll.currentToken)-1] == '\n' {
			ll.lineno--
			ll.column = ll.lastColumn

		} else {
			ll.column--
		}

		ll.currentToken = ll.currentToken[:len(ll.currentToken)-1]
		ll.currentTokenLen -= ll.lastTokenLen
	}
}

func (ll *lexer) Peek() (r rune) {
	r = ll.Next()
	ll.Back()
	return r
}

func (ll *lexer) Accept(r rune) (ok bool) {
	if ll.Next() == r {
		return true
	}

	ll.Back()
	return false
}

func (ll *lexer) AcceptOneOf(runes ...rune) (ok bool) {
	c := ll.Next()

	for _, r := range runes {
		if r == c {
			return true
		}
	}

	ll.Back()
	return false
}

func (ll *lexer) AcceptRun(f func(rune) bool) {
	for f(ll.Next()) {
	}

	ll.Back()
}

func (ll *lexer) GetToken() (s string) {
	buf := make([]byte, ll.currentTokenLen)
	pos := 0

	for _, r := range ll.currentToken {
		pos += utf8.EncodeRune(buf[pos:], r)
	}

	ll.Discard()
	return string(buf)
}

func (ll *lexer) Discard() {
	ll.currentToken = nil
	ll.currentTokenLen = 0
}

func ParseSquirtle(r io.Reader, tripleChan_ chan *Triple, errChan_ chan error, prefixMap_ map[string]string) {
	sqtlParserMutex.Lock()
	defer sqtlParserMutex.Unlock()

	/*
	   defer func() {
	       if err := recover(); err != nil {
	           if err != "foobar" {
	               panic(err)
	           }
	       }
	   }()
	*/

	defer close(tripleChan_)
	defer close(errChan_)

	sqtlTripleChan = tripleChan_
	sqtlErrChan = errChan_
	sqtlPrefixMap = prefixMap_
	sqtlNames = make(map[string]string)
	sqtlTemplates = make(map[string]*sqtlTemplate)
	sqtlStack = make([]sqtlStackEntry, 0)

	sqtlParse(newLexer(r))

	sqtlTripleChan = nil
	sqtlErrChan = nil
	sqtlPrefixMap = nil
	sqtlNames = nil
	sqtlTemplates = nil
	sqtlStack = nil
}

//line yacctab:1
var sqtlExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 65,
	26, 23,
	-2, 31,
	-1, 70,
	16, 32,
	18, 32,
	23, 32,
	-2, 44,
	-1, 71,
	16, 33,
	18, 33,
	23, 33,
	-2, 45,
	-1, 73,
	16, 34,
	18, 34,
	23, 34,
	-2, 47,
}

const sqtlNprod = 78
const sqtlPrivate = 57344

var sqtlTokenNames []string
var sqtlStates []string

const sqtlLast = 132

var sqtlAct = []int{

	69, 70, 16, 31, 16, 73, 18, 45, 18, 53,
	49, 38, 41, 40, 49, 41, 91, 95, 25, 93,
	89, 22, 25, 94, 92, 22, 65, 52, 21, 97,
	66, 33, 21, 99, 50, 47, 90, 85, 50, 48,
	34, 25, 35, 64, 22, 43, 47, 33, 29, 37,
	48, 63, 67, 62, 36, 82, 20, 78, 79, 42,
	25, 81, 25, 22, 77, 22, 88, 32, 12, 76,
	20, 80, 21, 44, 26, 3, 25, 10, 27, 22,
	7, 9, 74, 6, 14, 6, 21, 55, 56, 5,
	96, 4, 2, 98, 57, 54, 100, 1, 87, 58,
	59, 20, 86, 60, 61, 68, 11, 25, 10, 19,
	22, 13, 9, 71, 17, 14, 17, 21, 46, 28,
	30, 75, 8, 15, 8, 72, 84, 51, 83, 39,
	24, 23,
}
var sqtlPact = []int{

	95, -1000, 64, -1000, -1000, -1000, -1000, -1000, -1000, 48,
	29, 24, 42, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	37, -1000, -1000, -1000, -1000, -19, -1000, -1000, 54, -1000,
	-1000, -1000, 33, 10, -1000, -1000, 2, -1000, 83, -16,
	-1000, 83, 31, 1, 6, -1000, 50, -1000, -1000, -1000,
	-1000, 8, 15, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, -1000, 50, -1000, -1000, -7, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, 7, -1000, -1000, -1000,
	-1000, -1000, -1000, -2, -8, -1000, -3, -10, -1000, 50,
	17, 48, -1000, 11, -1000, 50, -1000, -1000, -1000, -1000,
	-1000,
}
var sqtlPgo = []int{

	0, 9, 131, 109, 130, 13, 129, 128, 127, 126,
	121, 113, 82, 1, 125, 123, 0, 118, 111, 106,
	5, 105, 102, 98, 97, 92, 75, 91, 89, 80,
	3, 73, 68, 67, 7,
}
var sqtlR1 = []int{

	0, 24, 25, 25, 26, 26, 26, 26, 26, 28,
	28, 27, 12, 30, 29, 32, 8, 8, 7, 7,
	9, 9, 22, 22, 23, 23, 33, 33, 10, 19,
	18, 18, 15, 15, 15, 31, 31, 34, 17, 17,
	17, 17, 21, 21, 16, 16, 16, 16, 16, 16,
	11, 14, 14, 14, 14, 14, 14, 14, 14, 20,
	13, 3, 3, 3, 3, 2, 4, 6, 6, 5,
	1, 1, 1, 1, 1, 1, 1, 1,
}
var sqtlR2 = []int{

	0, 2, 2, 1, 1, 1, 1, 1, 1, 2,
	2, 4, 2, 3, 4, 1, 3, 0, 1, 0,
	3, 1, 1, 0, 3, 1, 1, 1, 6, 1,
	1, 0, 1, 1, 1, 2, 1, 2, 1, 1,
	1, 1, 3, 1, 1, 1, 1, 1, 1, 1,
	2, 1, 3, 3, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 3, 2, 2, 1, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
}
var sqtlChk = []int{

	-1000, -24, -25, -26, -27, -28, -12, -29, -10, 17,
	13, -19, -32, -18, 20, -15, -13, -11, -20, -3,
	6, 22, 15, -2, -4, 12, 10, -26, -3, 19,
	-3, -30, -33, 23, 16, 18, 12, 12, 30, -6,
	-5, 31, 5, 12, -31, -34, -17, -13, -20, 4,
	28, -8, 25, -1, 12, 4, 5, 11, 16, 17,
	20, 21, -5, -1, 12, 25, 24, -34, -21, -16,
	-13, -11, -14, -20, -12, -10, 19, 14, 7, 8,
	21, 11, -30, -7, -9, 22, -22, -23, -16, 27,
	29, 9, 26, 27, 26, 27, -16, 12, -13, 22,
	-16,
}
var sqtlDef = []int{

	31, -2, 31, 3, 4, 5, 6, 7, 8, 0,
	0, 0, 0, 29, 15, 30, 32, 33, 34, 60,
	0, 59, 61, 62, 63, 64, 1, 2, 0, 9,
	10, 12, 0, 0, 26, 27, 17, 50, 0, 66,
	68, 0, 0, 0, 0, 36, 31, 38, 39, 40,
	41, 0, 19, 65, 70, 71, 72, 73, 74, 75,
	76, 77, 67, 69, 11, -2, 13, 35, 37, 43,
	-2, -2, 46, -2, 48, 49, 51, 54, 55, 56,
	57, 58, 14, 0, 18, 21, 0, 22, 25, 31,
	0, 0, 16, 0, 28, 31, 42, 52, 53, 20,
	24,
}
var sqtlTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	25, 26, 28, 3, 27, 3, 3, 31, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 30, 3,
	3, 3, 3, 3, 29, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 23, 3, 24,
}
var sqtlTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22,
}
var sqtlTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var sqtlDebug = 0

type sqtlLexer interface {
	Lex(lval *sqtlSymType) int
	Error(s string)
}

const sqtlFlag = -1000

func sqtlTokname(c int) string {
	if c > 0 && c <= len(sqtlToknames) {
		if sqtlToknames[c-1] != "" {
			return sqtlToknames[c-1]
		}
	}
	return fmt.Sprintf("tok-%v", c)
}

func sqtlStatname(s int) string {
	if s >= 0 && s < len(sqtlStatenames) {
		if sqtlStatenames[s] != "" {
			return sqtlStatenames[s]
		}
	}
	return fmt.Sprintf("state-%v", s)
}

func sqtllex1(lex sqtlLexer, lval *sqtlSymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = sqtlTok1[0]
		goto out
	}
	if char < len(sqtlTok1) {
		c = sqtlTok1[char]
		goto out
	}
	if char >= sqtlPrivate {
		if char < sqtlPrivate+len(sqtlTok2) {
			c = sqtlTok2[char-sqtlPrivate]
			goto out
		}
	}
	for i := 0; i < len(sqtlTok3); i += 2 {
		c = sqtlTok3[i+0]
		if c == char {
			c = sqtlTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = sqtlTok2[1] /* unknown char */
	}
	if sqtlDebug >= 3 {
		fmt.Printf("lex %U %s\n", uint(char), sqtlTokname(c))
	}
	return c
}

func sqtlParse(sqtllex sqtlLexer) int {
	var sqtln int
	var sqtllval sqtlSymType
	var sqtlVAL sqtlSymType
	sqtlS := make([]sqtlSymType, sqtlMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	sqtlstate := 0
	sqtlchar := -1
	sqtlp := -1
	goto sqtlstack

ret0:
	return 0

ret1:
	return 1

sqtlstack:
	/* put a state and value onto the stack */
	if sqtlDebug >= 4 {
		fmt.Printf("char %v in %v\n", sqtlTokname(sqtlchar), sqtlStatname(sqtlstate))
	}

	sqtlp++
	if sqtlp >= len(sqtlS) {
		nyys := make([]sqtlSymType, len(sqtlS)*2)
		copy(nyys, sqtlS)
		sqtlS = nyys
	}
	sqtlS[sqtlp] = sqtlVAL
	sqtlS[sqtlp].yys = sqtlstate

sqtlnewstate:
	sqtln = sqtlPact[sqtlstate]
	if sqtln <= sqtlFlag {
		goto sqtldefault /* simple state */
	}
	if sqtlchar < 0 {
		sqtlchar = sqtllex1(sqtllex, &sqtllval)
	}
	sqtln += sqtlchar
	if sqtln < 0 || sqtln >= sqtlLast {
		goto sqtldefault
	}
	sqtln = sqtlAct[sqtln]
	if sqtlChk[sqtln] == sqtlchar { /* valid shift */
		sqtlchar = -1
		sqtlVAL = sqtllval
		sqtlstate = sqtln
		if Errflag > 0 {
			Errflag--
		}
		goto sqtlstack
	}

sqtldefault:
	/* default state action */
	sqtln = sqtlDef[sqtlstate]
	if sqtln == -2 {
		if sqtlchar < 0 {
			sqtlchar = sqtllex1(sqtllex, &sqtllval)
		}

		/* look through exception table */
		xi := 0
		for {
			if sqtlExca[xi+0] == -1 && sqtlExca[xi+1] == sqtlstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			sqtln = sqtlExca[xi+0]
			if sqtln < 0 || sqtln == sqtlchar {
				break
			}
		}
		sqtln = sqtlExca[xi+1]
		if sqtln < 0 {
			goto ret0
		}
	}
	if sqtln == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			sqtllex.Error("syntax error")
			Nerrs++
			if sqtlDebug >= 1 {
				fmt.Printf("%s", sqtlStatname(sqtlstate))
				fmt.Printf("saw %s\n", sqtlTokname(sqtlchar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for sqtlp >= 0 {
				sqtln = sqtlPact[sqtlS[sqtlp].yys] + sqtlErrCode
				if sqtln >= 0 && sqtln < sqtlLast {
					sqtlstate = sqtlAct[sqtln] /* simulate a shift of "error" */
					if sqtlChk[sqtlstate] == sqtlErrCode {
						goto sqtlstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if sqtlDebug >= 2 {
					fmt.Printf("error recovery pops state %d\n", sqtlS[sqtlp].yys)
				}
				sqtlp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if sqtlDebug >= 2 {
				fmt.Printf("error recovery discards %s\n", sqtlTokname(sqtlchar))
			}
			if sqtlchar == sqtlEofCode {
				goto ret1
			}
			sqtlchar = -1
			goto sqtlnewstate /* try again in the same state */
		}
	}

	/* reduction by production sqtln */
	if sqtlDebug >= 2 {
		fmt.Printf("reduce %v in:\n\t%v\n", sqtln, sqtlStatname(sqtlstate))
	}

	sqtlnt := sqtln
	sqtlpt := sqtlp
	_ = sqtlpt // guard against "declared and not used"

	sqtlp -= sqtlR2[sqtln]
	sqtlVAL = sqtlS[sqtlp+1]

	/* consult goto table to find next state */
	sqtln = sqtlR1[sqtln]
	sqtlg := sqtlPgo[sqtln]
	sqtlj := sqtlg + sqtlS[sqtlp].yys + 1

	if sqtlj >= sqtlLast {
		sqtlstate = sqtlAct[sqtlg]
	} else {
		sqtlstate = sqtlAct[sqtlj]
		if sqtlChk[sqtlstate] != -sqtln {
			sqtlstate = sqtlAct[sqtlg]
		}
	}
	// dummy call; replaced with literal code
	switch sqtlnt {

	case 1:
		//line src/github.com/kierdavis/argo/squirtleparser.y:90
		{
			return 0
		}
	case 9:
		//line src/github.com/kierdavis/argo/squirtleparser.y:102
		{
			f, err := os.Open(sqtlS[sqtlpt-0].s)
			if err != nil {
				sqtlErrChan <- err
				return 1
			}

			n := sqtlParse(newLexer(f))
			f.Close()

			if n != 0 {
				return n
			}
		}
	case 10:
		//line src/github.com/kierdavis/argo/squirtleparser.y:118
		{
			resp, err := http.Get(sqtlS[sqtlpt-0].s)
			if err != nil {
				sqtlErrChan <- err
				return 1
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				sqtlErrChan <- fmt.Errorf("HTTP request returned %s", resp.Status)
				return 1
			}

			n := sqtlParse(newLexer(resp.Body))
			resp.Body.Close()

			if n != 0 {
				return n
			}
		}
	case 11:
		//line src/github.com/kierdavis/argo/squirtleparser.y:138
		{
			sqtlNames[sqtlS[sqtlpt-0].s] = sqtlS[sqtlpt-2].s
			sqtlPrefixMap[sqtlS[sqtlpt-2].s] = sqtlS[sqtlpt-0].s
		}
	case 12:
		//line src/github.com/kierdavis/argo/squirtleparser.y:140
		{
			sqtlVAL.t = sqtlS[sqtlpt-1].t
			sqtlStack = sqtlStack[:len(sqtlStack)-1]
		}
	case 14:
		//line src/github.com/kierdavis/argo/squirtleparser.y:145
		{
			sqtlTemplates[sqtlS[sqtlpt-2].s] = &sqtlTemplate{
				ArgNames: sqtlS[sqtlpt-1].sL,
				Triples:  sqtlStack[len(sqtlStack)-1].Template,
			}

			sqtlStack = sqtlStack[:len(sqtlStack)-1]
		}
	case 15:
		//line src/github.com/kierdavis/argo/squirtleparser.y:154
		{
			sqtlStack = append(sqtlStack, sqtlStackEntry{1, nil, []*Triple{}})
		}
	case 16:
		//line src/github.com/kierdavis/argo/squirtleparser.y:156
		{
			sqtlVAL.sL = sqtlS[sqtlpt-1].sL
		}
	case 17:
		//line src/github.com/kierdavis/argo/squirtleparser.y:157
		{
			sqtlVAL.sL = []string{}
		}
	case 18:
		//line src/github.com/kierdavis/argo/squirtleparser.y:159
		{
			sqtlVAL.sL = sqtlS[sqtlpt-0].sL
		}
	case 19:
		//line src/github.com/kierdavis/argo/squirtleparser.y:160
		{
			sqtlVAL.sL = []string{}
		}
	case 20:
		//line src/github.com/kierdavis/argo/squirtleparser.y:162
		{
			sqtlVAL.sL = append(sqtlS[sqtlpt-2].sL, sqtlS[sqtlpt-0].s)
		}
	case 21:
		//line src/github.com/kierdavis/argo/squirtleparser.y:163
		{
			sqtlVAL.sL = []string{sqtlS[sqtlpt-0].s}
		}
	case 22:
		//line src/github.com/kierdavis/argo/squirtleparser.y:165
		{
			sqtlVAL.tL = sqtlS[sqtlpt-0].tL
		}
	case 23:
		//line src/github.com/kierdavis/argo/squirtleparser.y:166
		{
			sqtlVAL.tL = []Term{}
		}
	case 24:
		//line src/github.com/kierdavis/argo/squirtleparser.y:168
		{
			sqtlVAL.tL = append(sqtlS[sqtlpt-2].tL, sqtlS[sqtlpt-0].t)
		}
	case 25:
		//line src/github.com/kierdavis/argo/squirtleparser.y:169
		{
			sqtlVAL.tL = []Term{sqtlS[sqtlpt-0].t}
		}
	case 28:
		//line src/github.com/kierdavis/argo/squirtleparser.y:175
		{
			sqtlVAL.t = sqtlS[sqtlpt-5].t
			templateName := sqtlS[sqtlpt-3].s
			args := sqtlS[sqtlpt-1].tL

			template := sqtlTemplates[templateName]
			if template == nil {
				sqtlErrChan <- fmt.Errorf("Undefined template: %s", template)
				return 1
			}

			bindings := make(map[string]Term)

			if len(template.ArgNames) != len(args) {
				sqtlErrChan <- fmt.Errorf("Wrong number of arguments for template %s: expected %d, got %d", templateName, len(template.ArgNames), len(args))
				return 1
			}

			for i, argName := range template.ArgNames {
				bindings[argName] = args[i]
			}

			for _, t := range template.Triples {
				subj := t.Subject
				pred := t.Predicate
				obj := t.Object

				if subj == nil {
					subj = sqtlVAL.t
				}

				if v, ok := subj.(*sqtlVar); ok {
					subj = bindings[v.Name]
				}

				if v, ok := pred.(*sqtlVar); ok {
					pred = bindings[v.Name]
				}

				if v, ok := obj.(*sqtlVar); ok {
					obj = bindings[v.Name]
				}

				sqtlTripleChan <- NewTriple(subj, pred, obj)
			}
		}
	case 29:
		//line src/github.com/kierdavis/argo/squirtleparser.y:223
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t

			var template []*Triple
			if len(sqtlStack) > 0 {
				template = sqtlStack[len(sqtlStack)-1].Template
			}

			sqtlStack = append(sqtlStack, sqtlStackEntry{1, sqtlVAL.t, template})
		}
	case 30:
		//line src/github.com/kierdavis/argo/squirtleparser.y:234
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 31:
		//line src/github.com/kierdavis/argo/squirtleparser.y:235
		{
			sqtlVAL.t = NewAnonNode()
		}
	case 32:
		//line src/github.com/kierdavis/argo/squirtleparser.y:237
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 33:
		//line src/github.com/kierdavis/argo/squirtleparser.y:238
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 34:
		//line src/github.com/kierdavis/argo/squirtleparser.y:239
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 37:
		//line src/github.com/kierdavis/argo/squirtleparser.y:245
		{
			top := sqtlStack[len(sqtlStack)-1]
			subj := top.Subject
			pred := sqtlS[sqtlpt-1].t

			for _, obj := range sqtlS[sqtlpt-0].tL {
				triple := NewTriple(subj, pred, obj)

				if top.Template != nil {
					top.Template = append(top.Template, triple)
				} else {
					sqtlTripleChan <- triple
				}
			}

			sqtlStack[len(sqtlStack)-1] = top
		}
	case 38:
		//line src/github.com/kierdavis/argo/squirtleparser.y:263
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 39:
		//line src/github.com/kierdavis/argo/squirtleparser.y:264
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 40:
		//line src/github.com/kierdavis/argo/squirtleparser.y:265
		{
			sqtlVAL.t = A
		}
	case 41:
		//line src/github.com/kierdavis/argo/squirtleparser.y:266
		{
			sqtlVAL.t = RDF.Get(fmt.Sprintf("_%d", sqtlStack[len(sqtlStack)-1].NextItem))
			sqtlStack[len(sqtlStack)-1].NextItem++
		}
	case 42:
		//line src/github.com/kierdavis/argo/squirtleparser.y:268
		{
			sqtlVAL.tL = append(sqtlS[sqtlpt-2].tL, sqtlS[sqtlpt-0].t)
		}
	case 43:
		//line src/github.com/kierdavis/argo/squirtleparser.y:269
		{
			sqtlVAL.tL = []Term{sqtlS[sqtlpt-0].t}
		}
	case 44:
		//line src/github.com/kierdavis/argo/squirtleparser.y:271
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 45:
		//line src/github.com/kierdavis/argo/squirtleparser.y:272
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 46:
		//line src/github.com/kierdavis/argo/squirtleparser.y:273
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 47:
		//line src/github.com/kierdavis/argo/squirtleparser.y:274
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 48:
		//line src/github.com/kierdavis/argo/squirtleparser.y:275
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 49:
		//line src/github.com/kierdavis/argo/squirtleparser.y:276
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 50:
		//line src/github.com/kierdavis/argo/squirtleparser.y:278
		{
			sqtlVAL.t = NewBlankNode(sqtlS[sqtlpt-0].s)
		}
	case 51:
		//line src/github.com/kierdavis/argo/squirtleparser.y:280
		{
			sqtlVAL.t = NewLiteral(sqtlS[sqtlpt-0].s)
		}
	case 52:
		//line src/github.com/kierdavis/argo/squirtleparser.y:281
		{
			sqtlVAL.t = NewLiteralWithLanguage(sqtlS[sqtlpt-2].s, sqtlS[sqtlpt-0].s)
		}
	case 53:
		//line src/github.com/kierdavis/argo/squirtleparser.y:282
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-2].s, sqtlS[sqtlpt-0].t)
		}
	case 54:
		//line src/github.com/kierdavis/argo/squirtleparser.y:283
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("integer"))
		}
	case 55:
		//line src/github.com/kierdavis/argo/squirtleparser.y:284
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("decimal"))
		}
	case 56:
		//line src/github.com/kierdavis/argo/squirtleparser.y:285
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("double"))
		}
	case 57:
		//line src/github.com/kierdavis/argo/squirtleparser.y:286
		{
			sqtlVAL.t = NewLiteralWithDatatype("true", XSD.Get("boolean"))
		}
	case 58:
		//line src/github.com/kierdavis/argo/squirtleparser.y:287
		{
			sqtlVAL.t = NewLiteralWithDatatype("false", XSD.Get("boolean"))
		}
	case 59:
		//line src/github.com/kierdavis/argo/squirtleparser.y:289
		{
			sqtlVAL.t = &sqtlVar{sqtlS[sqtlpt-0].s}
		}
	case 60:
		//line src/github.com/kierdavis/argo/squirtleparser.y:291
		{
			sqtlVAL.t = NewResource(sqtlS[sqtlpt-0].s)
		}
	case 61:
		//line src/github.com/kierdavis/argo/squirtleparser.y:293
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 62:
		//line src/github.com/kierdavis/argo/squirtleparser.y:294
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 63:
		//line src/github.com/kierdavis/argo/squirtleparser.y:295
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 64:
		//line src/github.com/kierdavis/argo/squirtleparser.y:296
		{
			sqtlVAL.s = getName(sqtlS[sqtlpt-0].s)
		}
	case 65:
		//line src/github.com/kierdavis/argo/squirtleparser.y:298
		{
			sqtlVAL.s = addHash(getName(sqtlS[sqtlpt-2].s)) + sqtlS[sqtlpt-0].s
		}
	case 66:
		//line src/github.com/kierdavis/argo/squirtleparser.y:300
		{
			sqtlVAL.s = stripSlash(getName(sqtlS[sqtlpt-1].s)) + sqtlS[sqtlpt-0].s
		}
	case 67:
		//line src/github.com/kierdavis/argo/squirtleparser.y:302
		{
			sqtlVAL.s = sqtlS[sqtlpt-1].s + sqtlS[sqtlpt-0].s
		}
	case 68:
		//line src/github.com/kierdavis/argo/squirtleparser.y:303
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 69:
		//line src/github.com/kierdavis/argo/squirtleparser.y:305
		{
			sqtlVAL.s = "/" + sqtlS[sqtlpt-0].s
		}
	case 70:
		//line src/github.com/kierdavis/argo/squirtleparser.y:307
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 71:
		//line src/github.com/kierdavis/argo/squirtleparser.y:308
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 72:
		//line src/github.com/kierdavis/argo/squirtleparser.y:309
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 73:
		//line src/github.com/kierdavis/argo/squirtleparser.y:310
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 74:
		sqtlVAL.s = sqtlS[sqtlpt-0].s
	case 75:
		//line src/github.com/kierdavis/argo/squirtleparser.y:312
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 76:
		sqtlVAL.s = sqtlS[sqtlpt-0].s
	case 77:
		//line src/github.com/kierdavis/argo/squirtleparser.y:314
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	}
	goto sqtlstack /* stack new state and value */
}
