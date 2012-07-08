//line src/github.com/kierdavis/argo/squirtle/parser.y:2

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

package squirtle

import (
	"bufio"
	"fmt"
	"github.com/kierdavis/argo"
	"io"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type stackEntry struct {
	Subject  argo.Term
	NextItem int
}

var LogParseMsg func(string)

var parserMutex sync.Mutex

var tripleChan chan *argo.Triple
var errChan chan error
var prefixMap map[string]string

var names map[string]string
var stack []stackEntry

//line src/github.com/kierdavis/argo/squirtle/parser.y:52
type yySymType struct {
	yys int
	s   string
	t   argo.Term
	tL  []argo.Term
}

const A = 57346
const AS = 57347
const BNODE = 57348
const DT = 57349
const EOF = 57350
const IDENTIFIER = 57351
const IRIREF = 57352
const NAME = 57353
const STRING = 57354

var yyToknames = []string{
	"A",
	"AS",
	"BNODE",
	"DT",
	"EOF",
	"IDENTIFIER",
	"IRIREF",
	"NAME",
	"STRING",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line src/github.com/kierdavis/argo/squirtle/parser.y:137

func getName(name string) (uri string) {
	uri, ok := names[name]
	if ok {
		return uri
	}

	if LogParseMsg != nil {
		LogParseMsg("Looking up prefix '" + name + "'")
	}

	uri, err := argo.LookupPrefix(name)
	if err == nil {
		names[name] = uri
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
	errChan <- fmt.Errorf("Syntax error: %s (at line %d col %d)", s, ll.lineno, ll.column)
	panic("foobar")
}

func (ll *lexer) Lex(lval *yySymType) (t int) {
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
			return A
		case "as":
			return AS
		case "name":
			return NAME
		}

		return IDENTIFIER

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

func ParseSquirtle(r io.Reader, tripleChan_ chan *argo.Triple, errChan_ chan error, prefixMap_ map[string]string) {
	parserMutex.Lock()
	defer parserMutex.Unlock()

	defer func() {
		if err := recover(); err != nil {
			if err != "foobar" {
				panic(err)
			}
		}
	}()

	defer close(tripleChan_)
	defer close(errChan_)

	tripleChan = tripleChan_
	errChan = errChan_
	prefixMap = prefixMap_
	names = make(map[string]string)
	stack = make([]stackEntry, 0)

	yyParse(newLexer(r))

	tripleChan = nil
	errChan = nil
	prefixMap = nil
	names = nil
	stack = nil
}

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 46,
	13, 11,
	-2, 21,
	-1, 48,
	13, 10,
	-2, 23,
}

const yyNprod = 43
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 76

var yyAct = []int{

	9, 5, 25, 10, 45, 32, 29, 23, 26, 34,
	16, 13, 26, 53, 32, 42, 33, 21, 51, 16,
	13, 55, 31, 52, 41, 33, 22, 39, 27, 31,
	3, 48, 49, 18, 46, 43, 40, 12, 16, 13,
	16, 13, 12, 50, 17, 16, 13, 6, 28, 20,
	4, 2, 48, 49, 56, 46, 54, 36, 37, 1,
	44, 7, 35, 12, 38, 11, 16, 13, 6, 8,
	30, 47, 19, 24, 15, 14,
}
var yyPact = []int{

	57, -1000, 36, -1000, -1000, -1000, 29, 4, -1000, -1000,
	-1000, -1000, 17, -1000, -1000, -1000, -11, -1000, -1000, 23,
	-1000, 10, -1000, 53, -7, -1000, 53, 15, 1, -1000,
	31, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, -1000, -1000, 2, -1000, -1000, -1000, -1000, -1000,
	6, 31, 12, 29, -1000, -1000, -1000,
}
var yyPgo = []int{

	0, 9, 75, 65, 74, 2, 73, 3, 1, 0,
	71, 4, 70, 69, 61, 60, 59, 51, 30, 50,
	49, 48, 6,
}
var yyR1 = []int{

	0, 16, 17, 17, 18, 18, 19, 8, 20, 14,
	13, 13, 13, 21, 21, 22, 12, 12, 12, 15,
	15, 11, 11, 11, 11, 7, 10, 10, 10, 9,
	3, 3, 3, 3, 2, 4, 6, 6, 5, 1,
	1, 1, 1,
}
var yyR2 = []int{

	0, 2, 2, 1, 1, 1, 4, 2, 3, 1,
	1, 1, 0, 2, 1, 2, 1, 1, 1, 3,
	1, 1, 1, 1, 1, 2, 1, 3, 3, 1,
	1, 1, 1, 1, 3, 2, 2, 1, 2, 1,
	1, 1, 1,
}
var yyChk = []int{

	-1000, -16, -17, -18, -19, -8, 11, -14, -13, -9,
	-7, -3, 6, 10, -2, -4, 9, 8, -18, -3,
	-20, 13, 9, 18, -6, -5, 19, 5, -21, -22,
	-12, -9, 4, 15, -1, 9, 4, 5, 11, -5,
	-1, 9, 14, -22, -15, -11, -7, -10, -9, -8,
	12, 16, 17, 7, -11, 9, -9,
}
var yyDef = []int{

	12, -2, 12, 3, 4, 5, 0, 0, 9, 10,
	11, 29, 0, 30, 31, 32, 33, 1, 2, 0,
	7, 0, 25, 0, 35, 37, 0, 0, 0, 14,
	12, 16, 17, 18, 34, 39, 40, 41, 42, 36,
	38, 6, 8, 13, 15, 20, -2, 22, -2, 24,
	26, 12, 0, 0, 19, 27, 28,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 15, 3, 16, 3, 3, 19, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 18, 3,
	3, 3, 3, 3, 17, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 13, 3, 14,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	if c > 0 && c <= len(yyToknames) {
		if yyToknames[c-1] != "" {
			return yyToknames[c-1]
		}
	}
	return fmt.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return fmt.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		fmt.Printf("lex %U %s\n", uint(char), yyTokname(c))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		fmt.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				fmt.Printf("%s", yyStatname(yystate))
				fmt.Printf("saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					fmt.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				fmt.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		fmt.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:66
		{
			return 0
		}
	case 6:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:74
		{
			names[yyS[yypt-0].s] = yyS[yypt-2].s
			prefixMap[yyS[yypt-2].s] = yyS[yypt-0].s
		}
	case 7:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:76
		{
			yyVAL.t = yyS[yypt-1].t
			stack = stack[:len(stack)-1]
		}
	case 9:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:80
		{
			yyVAL.t = yyS[yypt-0].t
			stack = append(stack, stackEntry{yyVAL.t, 1})
		}
	case 10:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:82
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 11:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:83
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 12:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:84
		{
			yyVAL.t = argo.NewAnonNode()
		}
	case 15:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:90
		{
			subj := stack[len(stack)-1].Subject
			pred := yyS[yypt-1].t
			for _, obj := range yyS[yypt-0].tL {
				tripleChan <- argo.NewTriple(subj, pred, obj)
			}
		}
	case 16:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:98
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 17:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:99
		{
			yyVAL.t = argo.A
		}
	case 18:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:100
		{
			yyVAL.t = argo.RDF.Get(fmt.Sprintf("_%d", stack[len(stack)-1].NextItem))
			stack[len(stack)-1].NextItem++
		}
	case 19:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:102
		{
			yyVAL.tL = append(yyS[yypt-2].tL, yyS[yypt-0].t)
		}
	case 20:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:103
		{
			yyVAL.tL = []argo.Term{yyS[yypt-0].t}
		}
	case 21:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:105
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 22:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:106
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 23:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:107
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 24:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:108
		{
			yyVAL.t = yyS[yypt-0].t
		}
	case 25:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:110
		{
			yyVAL.t = argo.NewBlankNode(yyS[yypt-0].s)
		}
	case 26:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:112
		{
			yyVAL.t = argo.NewLiteral(yyS[yypt-0].s)
		}
	case 27:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:113
		{
			yyVAL.t = argo.NewLiteralWithLanguage(yyS[yypt-2].s, yyS[yypt-0].s)
		}
	case 28:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:114
		{
			yyVAL.t = argo.NewLiteralWithDatatype(yyS[yypt-2].s, yyS[yypt-0].t)
		}
	case 29:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:116
		{
			yyVAL.t = argo.NewResource(yyS[yypt-0].s)
		}
	case 30:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:118
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 31:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:119
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 32:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:120
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 33:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:121
		{
			yyVAL.s = getName(yyS[yypt-0].s)
		}
	case 34:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:123
		{
			yyVAL.s = addHash(getName(yyS[yypt-2].s)) + yyS[yypt-0].s
		}
	case 35:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:125
		{
			yyVAL.s = stripSlash(getName(yyS[yypt-1].s)) + yyS[yypt-0].s
		}
	case 36:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:127
		{
			yyVAL.s = yyS[yypt-1].s + yyS[yypt-0].s
		}
	case 37:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:128
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 38:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:130
		{
			yyVAL.s = "/" + yyS[yypt-0].s
		}
	case 39:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:132
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 40:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:133
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 41:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:134
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 42:
		//line src/github.com/kierdavis/argo/squirtle/parser.y:135
		{
			yyVAL.s = yyS[yypt-0].s
		}
	}
	goto yystack /* stack new state and value */
}
