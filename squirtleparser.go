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

//line src/github.com/kierdavis/argo/squirtleparser.y:72
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
const INTEGER = 57355
const IRIREF = 57356
const IS = 57357
const NAME = 57358
const NEW = 57359
const STRING = 57360
const TEMPLATE = 57361
const TRUE = 57362
const VAR = 57363

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

//line src/github.com/kierdavis/argo/squirtleparser.y:271

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
	-1, 0,
	22, 28,
	-2, 24,
	-1, 1,
	1, -1,
	-2, 0,
	-1, 2,
	22, 28,
	-2, 24,
	-1, 41,
	22, 28,
	-2, 24,
	-1, 65,
	15, 29,
	22, 29,
	-2, 41,
	-1, 66,
	15, 30,
	22, 30,
	-2, 42,
	-1, 68,
	15, 31,
	22, 31,
	-2, 44,
	-1, 81,
	22, 28,
	25, 20,
	-2, 24,
	-1, 82,
	22, 28,
	-2, 24,
	-1, 95,
	22, 28,
	-2, 24,
}

const sqtlNprod = 75
const sqtlPrivate = 57344

var sqtlTokenNames []string
var sqtlStates []string

const sqtlLast = 124

var sqtlAct = []int{

	64, 65, 15, 28, 15, 68, 17, 40, 17, 44,
	49, 34, 37, 36, 37, 84, 95, 24, 86, 21,
	82, 94, 85, 81, 47, 29, 20, 31, 61, 44,
	93, 42, 45, 80, 83, 43, 24, 24, 21, 21,
	91, 42, 32, 60, 48, 43, 20, 62, 59, 58,
	77, 33, 45, 19, 73, 74, 30, 18, 76, 24,
	72, 21, 51, 52, 38, 71, 27, 75, 20, 53,
	50, 10, 3, 54, 55, 26, 39, 56, 57, 6,
	4, 2, 89, 90, 19, 19, 92, 1, 25, 88,
	24, 24, 21, 21, 8, 8, 96, 13, 13, 20,
	20, 69, 5, 87, 5, 66, 16, 63, 16, 70,
	7, 9, 7, 12, 41, 14, 67, 11, 79, 46,
	78, 35, 23, 22,
}
var sqtlPact = []int{

	79, -1000, 78, -1000, -1000, -1000, -1000, -1000, 24, 3,
	44, 10, -1000, -1000, 27, -1000, -1000, -1000, -1000, 39,
	-1000, -1000, -1000, -1000, -18, -1000, -1000, 59, -1000, 25,
	0, 32, -1000, -1000, 58, -16, -1000, 58, 31, 5,
	-1000, 47, -1000, -1000, -1000, -1000, 3, 12, -1, -1000,
	-1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -6, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, 6, -1000, -1000, -1000, -1000, -1000, -1000, -3, -8,
	-1000, 47, 47, 28, 24, -1000, 9, -4, -10, -1000,
	-1000, -1000, -1000, -1000, -1000, 47, -1000,
}
var sqtlPgo = []int{

	0, 10, 123, 57, 122, 13, 121, 120, 119, 118,
	109, 117, 105, 101, 1, 116, 115, 0, 114, 113,
	111, 5, 107, 103, 89, 87, 81, 72, 80, 79,
	3, 76, 71, 7,
}
var sqtlR1 = []int{

	0, 25, 26, 26, 27, 27, 27, 27, 28, 13,
	30, 29, 32, 8, 8, 7, 7, 9, 9, 23,
	23, 24, 24, 11, 11, 10, 20, 19, 19, 16,
	16, 16, 31, 31, 33, 18, 18, 18, 18, 22,
	22, 17, 17, 17, 17, 17, 17, 12, 15, 15,
	15, 15, 15, 15, 15, 15, 21, 14, 3, 3,
	3, 3, 2, 4, 6, 6, 5, 1, 1, 1,
	1, 1, 1, 1, 1,
}
var sqtlR2 = []int{

	0, 2, 2, 1, 1, 1, 1, 1, 4, 2,
	3, 4, 1, 3, 0, 1, 0, 3, 1, 1,
	0, 3, 1, 2, 0, 6, 1, 1, 0, 1,
	1, 1, 2, 1, 2, 1, 1, 1, 1, 3,
	1, 1, 1, 1, 1, 1, 1, 2, 1, 3,
	3, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 3, 2, 2, 1, 2, 1, 1, 1,
	1, 1, 1, 1, 1,
}
var sqtlChk = []int{

	-1000, -25, -26, -27, -28, -13, -29, -10, 16, -20,
	-32, -11, -19, 19, -16, -14, -12, -21, -3, 6,
	21, 14, -2, -4, 12, 10, -27, -3, -30, 22,
	12, 17, 15, 12, 29, -6, -5, 30, 5, -31,
	-33, -18, -14, -21, 4, 27, -8, 24, 12, -1,
	12, 4, 5, 11, 15, 16, 19, 20, -5, -1,
	12, 23, -33, -22, -17, -14, -12, -15, -21, -13,
	-10, 18, 13, 7, 8, 20, 11, -30, -7, -9,
	21, 24, 26, 28, 9, 25, 26, -23, -24, -17,
	-17, 12, -14, 21, 25, 26, -17,
}
var sqtlDef = []int{

	-2, -2, -2, 3, 4, 5, 6, 7, 0, 0,
	0, 0, 26, 12, 27, 29, 30, 31, 57, 0,
	56, 58, 59, 60, 61, 1, 2, 0, 9, 0,
	14, 0, 23, 47, 0, 63, 65, 0, 0, 0,
	33, -2, 35, 36, 37, 38, 0, 16, 0, 62,
	67, 68, 69, 70, 71, 72, 73, 74, 64, 66,
	8, 10, 32, 34, 40, -2, -2, 43, -2, 45,
	46, 48, 51, 52, 53, 54, 55, 11, 0, 15,
	18, -2, -2, 0, 0, 13, 0, 0, 19, 22,
	39, 49, 50, 17, 25, -2, 21,
}
var sqtlTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	24, 25, 27, 3, 26, 3, 3, 30, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 29, 3,
	3, 3, 3, 3, 28, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 22, 3, 23,
}
var sqtlTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
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
		//line src/github.com/kierdavis/argo/squirtleparser.y:88
		{
			return 0
		}
	case 8:
		//line src/github.com/kierdavis/argo/squirtleparser.y:98
		{
			sqtlNames[sqtlS[sqtlpt-0].s] = sqtlS[sqtlpt-2].s
			sqtlPrefixMap[sqtlS[sqtlpt-2].s] = sqtlS[sqtlpt-0].s
		}
	case 9:
		//line src/github.com/kierdavis/argo/squirtleparser.y:100
		{
			sqtlVAL.t = sqtlS[sqtlpt-1].t
			sqtlStack = sqtlStack[:len(sqtlStack)-1]
		}
	case 11:
		//line src/github.com/kierdavis/argo/squirtleparser.y:105
		{
			sqtlTemplates[sqtlS[sqtlpt-2].s] = &sqtlTemplate{
				ArgNames: sqtlS[sqtlpt-1].sL,
				Triples:  sqtlStack[len(sqtlStack)-1].Template,
			}

			sqtlStack = sqtlStack[:len(sqtlStack)-1]
		}
	case 12:
		//line src/github.com/kierdavis/argo/squirtleparser.y:114
		{
			sqtlStack = append(sqtlStack, sqtlStackEntry{1, nil, []*Triple{}})
		}
	case 13:
		//line src/github.com/kierdavis/argo/squirtleparser.y:116
		{
			sqtlVAL.sL = sqtlS[sqtlpt-1].sL
		}
	case 14:
		//line src/github.com/kierdavis/argo/squirtleparser.y:117
		{
			sqtlVAL.sL = []string{}
		}
	case 15:
		//line src/github.com/kierdavis/argo/squirtleparser.y:119
		{
			sqtlVAL.sL = sqtlS[sqtlpt-0].sL
		}
	case 16:
		//line src/github.com/kierdavis/argo/squirtleparser.y:120
		{
			sqtlVAL.sL = []string{}
		}
	case 17:
		//line src/github.com/kierdavis/argo/squirtleparser.y:122
		{
			sqtlVAL.sL = append(sqtlS[sqtlpt-2].sL, sqtlS[sqtlpt-0].s)
		}
	case 18:
		//line src/github.com/kierdavis/argo/squirtleparser.y:123
		{
			sqtlVAL.sL = []string{sqtlS[sqtlpt-0].s}
		}
	case 19:
		//line src/github.com/kierdavis/argo/squirtleparser.y:125
		{
			sqtlVAL.tL = sqtlS[sqtlpt-0].tL
		}
	case 20:
		//line src/github.com/kierdavis/argo/squirtleparser.y:126
		{
			sqtlVAL.tL = []Term{}
		}
	case 21:
		//line src/github.com/kierdavis/argo/squirtleparser.y:128
		{
			sqtlVAL.tL = append(sqtlS[sqtlpt-2].tL, sqtlS[sqtlpt-0].t)
		}
	case 22:
		//line src/github.com/kierdavis/argo/squirtleparser.y:129
		{
			sqtlVAL.tL = []Term{sqtlS[sqtlpt-0].t}
		}
	case 23:
		//line src/github.com/kierdavis/argo/squirtleparser.y:131
		{
			sqtlVAL.t = sqtlS[sqtlpt-1].t
		}
	case 24:
		//line src/github.com/kierdavis/argo/squirtleparser.y:132
		{
			sqtlVAL.t = NewAnonNode()
		}
	case 25:
		//line src/github.com/kierdavis/argo/squirtleparser.y:135
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
	case 26:
		//line src/github.com/kierdavis/argo/squirtleparser.y:178
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t

			var template []*Triple
			if len(sqtlStack) > 0 {
				template = sqtlStack[len(sqtlStack)-1].Template
			}

			sqtlStack = append(sqtlStack, sqtlStackEntry{1, sqtlVAL.t, template})
		}
	case 27:
		//line src/github.com/kierdavis/argo/squirtleparser.y:189
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 28:
		//line src/github.com/kierdavis/argo/squirtleparser.y:190
		{
			sqtlVAL.t = NewAnonNode()
		}
	case 29:
		//line src/github.com/kierdavis/argo/squirtleparser.y:192
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 30:
		//line src/github.com/kierdavis/argo/squirtleparser.y:193
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 31:
		//line src/github.com/kierdavis/argo/squirtleparser.y:194
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 34:
		//line src/github.com/kierdavis/argo/squirtleparser.y:200
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
	case 35:
		//line src/github.com/kierdavis/argo/squirtleparser.y:218
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 36:
		//line src/github.com/kierdavis/argo/squirtleparser.y:219
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 37:
		//line src/github.com/kierdavis/argo/squirtleparser.y:220
		{
			sqtlVAL.t = A
		}
	case 38:
		//line src/github.com/kierdavis/argo/squirtleparser.y:221
		{
			sqtlVAL.t = RDF.Get(fmt.Sprintf("_%d", sqtlStack[len(sqtlStack)-1].NextItem))
			sqtlStack[len(sqtlStack)-1].NextItem++
		}
	case 39:
		//line src/github.com/kierdavis/argo/squirtleparser.y:223
		{
			sqtlVAL.tL = append(sqtlS[sqtlpt-2].tL, sqtlS[sqtlpt-0].t)
		}
	case 40:
		//line src/github.com/kierdavis/argo/squirtleparser.y:224
		{
			sqtlVAL.tL = []Term{sqtlS[sqtlpt-0].t}
		}
	case 41:
		//line src/github.com/kierdavis/argo/squirtleparser.y:226
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 42:
		//line src/github.com/kierdavis/argo/squirtleparser.y:227
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 43:
		//line src/github.com/kierdavis/argo/squirtleparser.y:228
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 44:
		//line src/github.com/kierdavis/argo/squirtleparser.y:229
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 45:
		//line src/github.com/kierdavis/argo/squirtleparser.y:230
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 46:
		//line src/github.com/kierdavis/argo/squirtleparser.y:231
		{
			sqtlVAL.t = sqtlS[sqtlpt-0].t
		}
	case 47:
		//line src/github.com/kierdavis/argo/squirtleparser.y:233
		{
			sqtlVAL.t = NewBlankNode(sqtlS[sqtlpt-0].s)
		}
	case 48:
		//line src/github.com/kierdavis/argo/squirtleparser.y:235
		{
			sqtlVAL.t = NewLiteral(sqtlS[sqtlpt-0].s)
		}
	case 49:
		//line src/github.com/kierdavis/argo/squirtleparser.y:236
		{
			sqtlVAL.t = NewLiteralWithLanguage(sqtlS[sqtlpt-2].s, sqtlS[sqtlpt-0].s)
		}
	case 50:
		//line src/github.com/kierdavis/argo/squirtleparser.y:237
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-2].s, sqtlS[sqtlpt-0].t)
		}
	case 51:
		//line src/github.com/kierdavis/argo/squirtleparser.y:238
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("integer"))
		}
	case 52:
		//line src/github.com/kierdavis/argo/squirtleparser.y:239
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("decimal"))
		}
	case 53:
		//line src/github.com/kierdavis/argo/squirtleparser.y:240
		{
			sqtlVAL.t = NewLiteralWithDatatype(sqtlS[sqtlpt-0].s, XSD.Get("double"))
		}
	case 54:
		//line src/github.com/kierdavis/argo/squirtleparser.y:241
		{
			sqtlVAL.t = NewLiteralWithDatatype("true", XSD.Get("boolean"))
		}
	case 55:
		//line src/github.com/kierdavis/argo/squirtleparser.y:242
		{
			sqtlVAL.t = NewLiteralWithDatatype("false", XSD.Get("boolean"))
		}
	case 56:
		//line src/github.com/kierdavis/argo/squirtleparser.y:244
		{
			sqtlVAL.t = &sqtlVar{sqtlS[sqtlpt-0].s}
		}
	case 57:
		//line src/github.com/kierdavis/argo/squirtleparser.y:246
		{
			sqtlVAL.t = NewResource(sqtlS[sqtlpt-0].s)
		}
	case 58:
		//line src/github.com/kierdavis/argo/squirtleparser.y:248
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 59:
		//line src/github.com/kierdavis/argo/squirtleparser.y:249
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 60:
		//line src/github.com/kierdavis/argo/squirtleparser.y:250
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 61:
		//line src/github.com/kierdavis/argo/squirtleparser.y:251
		{
			sqtlVAL.s = getName(sqtlS[sqtlpt-0].s)
		}
	case 62:
		//line src/github.com/kierdavis/argo/squirtleparser.y:253
		{
			sqtlVAL.s = addHash(getName(sqtlS[sqtlpt-2].s)) + sqtlS[sqtlpt-0].s
		}
	case 63:
		//line src/github.com/kierdavis/argo/squirtleparser.y:255
		{
			sqtlVAL.s = stripSlash(getName(sqtlS[sqtlpt-1].s)) + sqtlS[sqtlpt-0].s
		}
	case 64:
		//line src/github.com/kierdavis/argo/squirtleparser.y:257
		{
			sqtlVAL.s = sqtlS[sqtlpt-1].s + sqtlS[sqtlpt-0].s
		}
	case 65:
		//line src/github.com/kierdavis/argo/squirtleparser.y:258
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 66:
		//line src/github.com/kierdavis/argo/squirtleparser.y:260
		{
			sqtlVAL.s = "/" + sqtlS[sqtlpt-0].s
		}
	case 67:
		//line src/github.com/kierdavis/argo/squirtleparser.y:262
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 68:
		//line src/github.com/kierdavis/argo/squirtleparser.y:263
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 69:
		//line src/github.com/kierdavis/argo/squirtleparser.y:264
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 70:
		//line src/github.com/kierdavis/argo/squirtleparser.y:265
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 71:
		sqtlVAL.s = sqtlS[sqtlpt-0].s
	case 72:
		//line src/github.com/kierdavis/argo/squirtleparser.y:267
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	case 73:
		sqtlVAL.s = sqtlS[sqtlpt-0].s
	case 74:
		//line src/github.com/kierdavis/argo/squirtleparser.y:269
		{
			sqtlVAL.s = sqtlS[sqtlpt-0].s
		}
	}
	goto sqtlstack /* stack new state and value */
}
