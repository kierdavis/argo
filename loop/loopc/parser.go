//line src/github.com/kierdavis/go/loopc/parser.y:2
package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var nameMap = map[string]string{
	"rdf":  "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
	"rdfs": "http://www.w3.org/2000/01/rdf-schema#",
	"owl":  "http://www.w3.org/2002/07/owl#",
	"foaf": "http://xmlns.com/foaf/0.1/",
	"dc":   "http://purl.org/dc/elements/1.1/",
	"dct":  "http://purl.org/dc/terms/",
	"loop": "http://kierdavis.com/data/vocab/loop/",
	"xsd":  "http://www.w3.org/2001/XMLSchema#",
}

var currentFuncURI string
var currentVars map[string]string
var parserOutput = make(chan *Func, 2)

//line src/github.com/kierdavis/go/loopc/parser.y:29
type yySymType struct {
	yys    int
	lineno int
	column int
	name   string

	s  string
	i  int64
	f  float64
	A  *Arg
	AL []*Arg
	T  Type
	S  Statement
	SL []Statement
	E  Expression
	EM map[string]Expression
	PO *PredicateObjectPair
	O  Object
	L  *LiteralObject
}

const INTCONST = 57346
const FLOATCONST = 57347
const IDENTIFIER = 57348
const QUOTEDSTRING = 57349
const URIREF = 57350
const AS = 57351
const BOOLEAN = 57352
const DATA = 57353
const DTSYMBOL = 57354
const FALSE = 57355
const FLOAT = 57356
const FUNC = 57357
const INTEGER = 57358
const NAME = 57359
const OF = 57360
const RESOURCE = 57361
const STRING = 57362
const TRUE = 57363

var yyToknames = []string{
	"INTCONST",
	"FLOATCONST",
	"IDENTIFIER",
	"QUOTEDSTRING",
	"URIREF",
	"AS",
	"BOOLEAN",
	"DATA",
	"DTSYMBOL",
	"FALSE",
	"FLOAT",
	"FUNC",
	"INTEGER",
	"NAME",
	"OF",
	"RESOURCE",
	"STRING",
	"TRUE",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line src/github.com/kierdavis/go/loopc/parser.y:545

type yyLex struct {
	l *lexer
}

func (lex *yyLex) Lex(lval *yySymType) (t int) {
	tok := lex.l.lex()
	lval.lineno = tok.lineno
	lval.column = tok.column

	switch tok.typ {
	case tokError:
		fmt.Fprintln(os.Stderr, tok.val)
		fallthrough

	case tokEOF:
		return -1

	case tokIdentifier:
		lval.s = tok.val
		return IDENTIFIER

	case tokNumber:
		if strings.Index(tok.val, ".") >= 0 {
			n, err := strconv.ParseFloat(tok.val, 64)
			if err != nil {
				panic(err) // The lexer should have already ensured that it's a valid number.
			}

			lval.f = n
			return FLOATCONST

		} else {
			n, err := strconv.ParseInt(tok.val, 10, 64)
			if err != nil {
				panic(err) // The lexer should have already ensured that it's a valid number.
			}

			lval.i = n
			return INTCONST
		}

	case tokURIRef:
		lval.s = tok.val
		return URIREF

	case tokQuotedString:
		lval.s = tok.val
		return QUOTEDSTRING

	case tokDTSymbol:
		return DTSYMBOL

	case tokAs:
		return AS
	case tokBoolean:
		return BOOLEAN
	case tokData:
		return DATA
	case tokFalse:
		return FALSE
	case tokFloat:
		return FLOAT
	case tokFunc:
		return FUNC
	case tokInteger:
		return INTEGER
	case tokName:
		return NAME
	case tokOf:
		return OF
	case tokResource:
		return RESOURCE
	case tokString:
		return STRING
	case tokTrue:
		return TRUE
	}

	return int(tok.typ)
}

func (lex *yyLex) Error(s string) {
	fmt.Fprintf(os.Stderr, "[near line %d column %d] %s\n", lex.l.lineno, lex.l.column, s)
	os.Exit(1)
}

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
	-1, 82,
	25, 57,
	-2, 44,
	-1, 83,
	25, 56,
	-2, 45,
}

const yyNprod = 66
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 109

var yyAct = []int{

	32, 9, 72, 46, 31, 36, 29, 8, 15, 42,
	17, 20, 20, 10, 47, 11, 70, 71, 10, 49,
	11, 48, 44, 22, 39, 43, 16, 76, 77, 82,
	47, 11, 44, 45, 65, 41, 79, 10, 62, 11,
	10, 19, 11, 68, 78, 63, 64, 59, 7, 5,
	6, 39, 84, 53, 58, 66, 28, 54, 60, 55,
	25, 45, 56, 57, 81, 83, 85, 80, 73, 10,
	69, 11, 86, 10, 47, 11, 38, 26, 11, 24,
	23, 4, 3, 2, 1, 30, 61, 81, 83, 87,
	80, 73, 33, 74, 40, 27, 51, 50, 52, 21,
	34, 35, 13, 18, 75, 37, 12, 67, 14,
}
var yyPact = []int{

	-1000, -1000, 33, -1000, -1000, -1000, 63, 63, 17, -1000,
	-18, -1000, -1000, -1000, -2, -1000, 74, 73, -17, -1000,
	71, 34, 70, -1000, -1000, -1000, -1000, -1000, 63, -1000,
	-1000, -1000, 7, -1000, -5, -8, -1000, 43, -18, -1000,
	31, -1000, -1000, 52, 12, -1000, -1000, 22, -1000, 70,
	36, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	67, -10, -1000, 23, 46, 63, -1000, -1000, -1000, -1000,
	-1000, 63, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -3, -18, -1000, -1000, -1000, 23, -1000,
}
var yyPgo = []int{

	0, 108, 107, 106, 1, 105, 0, 104, 41, 103,
	102, 5, 101, 100, 99, 98, 97, 96, 6, 95,
	94, 93, 2, 92, 4, 86, 85, 9, 3, 84,
	83, 82, 81, 49,
}
var yyR1 = []int{

	0, 29, 30, 30, 31, 31, 32, 33, 1, 19,
	19, 20, 20, 18, 18, 26, 26, 27, 27, 28,
	28, 28, 22, 22, 24, 21, 21, 21, 21, 21,
	21, 23, 23, 25, 25, 14, 14, 13, 13, 12,
	12, 11, 5, 5, 7, 7, 16, 16, 17, 15,
	15, 15, 15, 15, 2, 2, 6, 6, 4, 4,
	4, 3, 10, 9, 9, 8,
}
var yyR2 = []int{

	0, 1, 0, 2, 1, 1, 4, 4, 1, 3,
	1, 2, 1, 1, 1, 2, 4, 1, 1, 1,
	3, 3, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 4, 3, 4, 2, 0, 3, 0, 1, 3,
	1, 3, 1, 1, 1, 1, 0, 1, 1, 1,
	1, 1, 1, 1, 0, 1, 1, 1, 1, 1,
	1, 3, 2, 2, 1, 2,
}
var yyChk = []int{

	-1000, -29, -30, -31, -32, -33, 17, 15, -6, -4,
	6, 8, -3, -10, -1, -6, 9, 28, -9, -8,
	29, -14, 25, 6, 6, -8, 6, -19, 22, -18,
	-26, -24, -6, -23, -13, -12, -11, -5, 6, -4,
	-20, -18, -27, 18, 25, -6, -28, 7, 26, 27,
	-16, -17, -15, 10, 14, 16, 19, 20, 23, -18,
	6, -25, 26, -6, 24, 12, -11, -2, 7, -27,
	26, 27, -22, -24, -21, -7, 4, 5, 21, 13,
	-28, -6, 6, -4, 6, -6, -6, -22,
}
var yyDef = []int{

	2, -2, 1, 3, 4, 5, 0, 0, 0, 56,
	57, 58, 59, 60, 35, 8, 0, 0, 62, 64,
	0, 0, 37, 6, 61, 63, 65, 7, 0, 10,
	13, 14, 0, 24, 0, 38, 40, 46, 42, 43,
	0, 12, 15, 0, 0, 17, 18, 19, 36, 0,
	54, 47, 48, 49, 50, 51, 52, 53, 9, 11,
	0, 0, 32, 0, 0, 0, 39, 41, 55, 16,
	31, 0, 34, 22, 23, 25, 26, 27, 28, 29,
	30, 0, -2, -2, 20, 21, 0, 33,
}
var yyTok1 = []int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	25, 26, 3, 3, 27, 3, 3, 29, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 28, 3,
	3, 3, 3, 3, 24, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 22, 3, 23,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
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
		//line src/github.com/kierdavis/go/loopc/parser.y:69
		{
			close(parserOutput)
		}
	case 6:
		//line src/github.com/kierdavis/go/loopc/parser.y:84
		{
			nameMap[yyS[yypt-0].s] = yyS[yypt-2].s
		}
	case 7:
		//line src/github.com/kierdavis/go/loopc/parser.y:91
		{
			parserOutput <- &Func{yyS[yypt-2].s, yyS[yypt-1].AL, yyS[yypt-0].SL}
			currentFuncURI = ""
		}
	case 8:
		//line src/github.com/kierdavis/go/loopc/parser.y:98
		{
			yyVAL.s = yyS[yypt-0].s
			currentFuncURI = yyVAL.s
			currentVars = make(map[string]string)

			last := currentFuncURI[len(currentFuncURI)-1]
			if last == '/' || last == '#' {
				currentFuncURI = currentFuncURI[:len(currentFuncURI)-1]
			}
		}
	case 9:
		//line src/github.com/kierdavis/go/loopc/parser.y:112
		{
			yyVAL.SL = yyS[yypt-1].SL
		}
	case 10:
		//line src/github.com/kierdavis/go/loopc/parser.y:117
		{
			yyVAL.SL = []Statement{yyS[yypt-0].S}
		}
	case 11:
		//line src/github.com/kierdavis/go/loopc/parser.y:124
		{
			yyVAL.SL = append(yyS[yypt-1].SL, yyS[yypt-0].S)
		}
	case 12:
		//line src/github.com/kierdavis/go/loopc/parser.y:129
		{
			yyVAL.SL = []Statement{yyS[yypt-0].S}
		}
	case 13:
		//line src/github.com/kierdavis/go/loopc/parser.y:136
		{
			yyVAL.S = yyS[yypt-0].PO
		}
	case 14:
		//line src/github.com/kierdavis/go/loopc/parser.y:141
		{
			yyVAL.S = yyS[yypt-0].E
		}
	case 15:
		//line src/github.com/kierdavis/go/loopc/parser.y:148
		{
			yyVAL.PO = &PredicateObjectPair{"", yyS[yypt-1].s, yyS[yypt-0].O}
		}
	case 16:
		//line src/github.com/kierdavis/go/loopc/parser.y:153
		{
			yyVAL.PO = &PredicateObjectPair{currentVars[yyS[yypt-1].s], yyS[yypt-3].s, yyS[yypt-0].O}
		}
	case 17:
		//line src/github.com/kierdavis/go/loopc/parser.y:160
		{
			yyVAL.O = ResourceObject(yyS[yypt-0].s)
		}
	case 18:
		//line src/github.com/kierdavis/go/loopc/parser.y:165
		{
			yyVAL.O = yyS[yypt-0].L
		}
	case 19:
		//line src/github.com/kierdavis/go/loopc/parser.y:172
		{
			yyVAL.L = &LiteralObject{yyS[yypt-0].s, "", ""}
		}
	case 20:
		//line src/github.com/kierdavis/go/loopc/parser.y:177
		{
			yyVAL.L = &LiteralObject{yyS[yypt-2].s, yyS[yypt-0].s, ""}
		}
	case 21:
		//line src/github.com/kierdavis/go/loopc/parser.y:182
		{
			yyVAL.L = &LiteralObject{yyS[yypt-2].s, "", yyS[yypt-0].s}
		}
	case 22:
		//line src/github.com/kierdavis/go/loopc/parser.y:189
		{
			yyVAL.E = yyS[yypt-0].E
		}
	case 23:
		//line src/github.com/kierdavis/go/loopc/parser.y:194
		{
			yyVAL.E = yyS[yypt-0].E
		}
	case 24:
		//line src/github.com/kierdavis/go/loopc/parser.y:201
		{
			yyVAL.E = yyS[yypt-0].E
		}
	case 25:
		//line src/github.com/kierdavis/go/loopc/parser.y:208
		{
			yyVAL.E = ResourceConstant(yyS[yypt-0].s)
		}
	case 26:
		//line src/github.com/kierdavis/go/loopc/parser.y:213
		{
			yyVAL.E = IntegerConstant(yyS[yypt-0].i)
		}
	case 27:
		//line src/github.com/kierdavis/go/loopc/parser.y:218
		{
			yyVAL.E = FloatConstant(yyS[yypt-0].f)
		}
	case 28:
		//line src/github.com/kierdavis/go/loopc/parser.y:223
		{
			yyVAL.E = BooleanConstant(true)
		}
	case 29:
		//line src/github.com/kierdavis/go/loopc/parser.y:228
		{
			yyVAL.E = BooleanConstant(false)
		}
	case 30:
		//line src/github.com/kierdavis/go/loopc/parser.y:233
		{
			lit := yyS[yypt-0].L

			switch lit.Datatype {
			case XSDboolean:
				switch lit.Value {
				case "true", "1":
					yyVAL.E = BooleanConstant(true)

				case "false", "0":
					yyVAL.E = BooleanConstant(false)

				default:
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for boolean constant: %s", 0, 0, lit.Value)
				}

			case XSDbase64Binary:
				data, err := base64.StdEncoding.DecodeString(lit.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for base64 constant: %s (%s)", 0, 0, lit.Value, err.Error())
				} else {
					yyVAL.E = DataConstant(data)
				}

			case XSDhexBinary:
				data, err := hex.DecodeString(lit.Value)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for hex constant: %s (%s)", 0, 0, lit.Value, err.Error())
				} else {
					yyVAL.E = DataConstant(data)
				}

			case XSDfloat, XSDdecimal, XSDdouble:
				n, err := strconv.ParseFloat(lit.Value, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for float constant: %s (%s)", 0, 0, lit.Value, err.Error())
				} else {
					yyVAL.E = FloatConstant(n)
				}

			case XSDinteger, XSDnonPositiveInteger, XSDnegativeInteger, XSDlong, XSDint, XSDshort, XSDbyte, XSDnonNegativeInteger, XSDunsignedLong, XSDunsignedInt, XSDunsignedShort, XSDunsignedByte, XSDpositiveInteger:
				n, err := strconv.ParseInt(lit.Value, 10, 64)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for integer constant: %s (%s)", 0, 0, lit.Value, err.Error())
				} else {
					yyVAL.E = IntegerConstant(n)
				}

			case XSDanyURI:
				yyVAL.E = ResourceConstant(lit.Value)

			case XSDQName:
				colonPos := strings.Index(lit.Value, ":")
				if colonPos < 0 {
					fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for QName constant: %s", 0, 0, lit.Value)
				} else {
					a := lit.Value[:colonPos]
					b := lit.Value[colonPos+1:]
					yyVAL.E = ResourceConstant(nameMap[a] + b)
				}

			default:
				yyVAL.E = StringConstant(lit.Value)
			}
		}
	case 31:
		//line src/github.com/kierdavis/go/loopc/parser.y:302
		{
			yyVAL.E = &FuncCall{yyS[yypt-3].s, yyS[yypt-1].EM}
		}
	case 32:
		//line src/github.com/kierdavis/go/loopc/parser.y:307
		{
			yyVAL.E = &FuncCall{yyS[yypt-2].s, map[string]Expression{}}
		}
	case 33:
		//line src/github.com/kierdavis/go/loopc/parser.y:314
		{
			yyVAL.EM = yyS[yypt-3].EM
			yyVAL.EM[yyS[yypt-1].s] = yyS[yypt-0].E
		}
	case 34:
		//line src/github.com/kierdavis/go/loopc/parser.y:320
		{
			yyVAL.EM = map[string]Expression{yyS[yypt-1].s: yyS[yypt-0].E}
		}
	case 35:
		//line src/github.com/kierdavis/go/loopc/parser.y:327
		{
			yyVAL.AL = []*Arg{}
		}
	case 36:
		//line src/github.com/kierdavis/go/loopc/parser.y:332
		{
			yyVAL.AL = yyS[yypt-1].AL
		}
	case 37:
		//line src/github.com/kierdavis/go/loopc/parser.y:339
		{
			yyVAL.AL = []*Arg{}
		}
	case 38:
		//line src/github.com/kierdavis/go/loopc/parser.y:344
		{
			yyVAL.AL = yyS[yypt-0].AL
		}
	case 39:
		//line src/github.com/kierdavis/go/loopc/parser.y:351
		{
			yyVAL.AL = append(yyS[yypt-2].AL, yyS[yypt-0].A)
		}
	case 40:
		//line src/github.com/kierdavis/go/loopc/parser.y:356
		{
			yyVAL.AL = []*Arg{yyS[yypt-0].A}
		}
	case 41:
		//line src/github.com/kierdavis/go/loopc/parser.y:363
		{
			yyVAL.A = &Arg{yyS[yypt-2].s, yyS[yypt-1].T, yyS[yypt-0].s}
		}
	case 42:
		//line src/github.com/kierdavis/go/loopc/parser.y:370
		{
			name := yyS[yypt-0].s
			uri, ok := nameMap[name]

			if ok {
				yyVAL.s = uri
			} else {
				yyVAL.s = currentFuncURI + "/arg/" + name
				currentVars[name] = yyVAL.s
			}
		}
	case 43:
		//line src/github.com/kierdavis/go/loopc/parser.y:383
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 44:
		//line src/github.com/kierdavis/go/loopc/parser.y:390
		{
			name := yyS[yypt-0].s
			uri, ok := currentVars[name]

			if ok {
				yyVAL.s = uri
			} else {
				yyVAL.s = nameMap[name]
			}
		}
	case 45:
		//line src/github.com/kierdavis/go/loopc/parser.y:402
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 46:
		//line src/github.com/kierdavis/go/loopc/parser.y:409
		{
			yyVAL.T = nil
		}
	case 47:
		//line src/github.com/kierdavis/go/loopc/parser.y:414
		{
			yyVAL.T = yyS[yypt-0].T
		}
	case 48:
		//line src/github.com/kierdavis/go/loopc/parser.y:421
		{
			yyVAL.T = yyS[yypt-0].T
		}
	case 49:
		//line src/github.com/kierdavis/go/loopc/parser.y:428
		{
			yyVAL.T = Boolean
		}
	case 50:
		//line src/github.com/kierdavis/go/loopc/parser.y:433
		{
			yyVAL.T = Float
		}
	case 51:
		//line src/github.com/kierdavis/go/loopc/parser.y:438
		{
			yyVAL.T = Integer
		}
	case 52:
		//line src/github.com/kierdavis/go/loopc/parser.y:443
		{
			yyVAL.T = Resource
		}
	case 53:
		//line src/github.com/kierdavis/go/loopc/parser.y:448
		{
			yyVAL.T = String
		}
	case 54:
		//line src/github.com/kierdavis/go/loopc/parser.y:455
		{
			yyVAL.s = ""
		}
	case 55:
		//line src/github.com/kierdavis/go/loopc/parser.y:460
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 56:
		//line src/github.com/kierdavis/go/loopc/parser.y:467
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 57:
		//line src/github.com/kierdavis/go/loopc/parser.y:472
		{
			yyVAL.s = nameMap[yyS[yypt-0].s]
		}
	case 58:
		//line src/github.com/kierdavis/go/loopc/parser.y:479
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 59:
		//line src/github.com/kierdavis/go/loopc/parser.y:484
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 60:
		//line src/github.com/kierdavis/go/loopc/parser.y:489
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 61:
		//line src/github.com/kierdavis/go/loopc/parser.y:496
		{
			base := nameMap[yyS[yypt-2].s]
			if len(base) > 0 {
				last := base[len(base)-1]
				if last != '/' && last != '#' {
					base += "#"
				}

			} else {
				base = "#"
			}

			yyVAL.s = base + yyS[yypt-0].s
		}
	case 62:
		//line src/github.com/kierdavis/go/loopc/parser.y:514
		{
			base := nameMap[yyS[yypt-1].s]
			last := base[len(base)-1]
			if last == '/' || last == '#' {
				base = base[:len(base)-1]
			}

			yyVAL.s = base + yyS[yypt-0].s
		}
	case 63:
		//line src/github.com/kierdavis/go/loopc/parser.y:527
		{
			yyVAL.s = yyS[yypt-1].s + yyS[yypt-0].s
		}
	case 64:
		//line src/github.com/kierdavis/go/loopc/parser.y:532
		{
			yyVAL.s = yyS[yypt-0].s
		}
	case 65:
		//line src/github.com/kierdavis/go/loopc/parser.y:539
		{
			yyVAL.s = "/" + yyS[yypt-0].s
		}
	}
	goto yystack /* stack new state and value */
}
