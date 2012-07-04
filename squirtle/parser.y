%{
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
        Subject argo.Term
        NextItem int
    }
    
    var parserMutex sync.Mutex
    
    var tripleChan chan *argo.Triple
    var errChan chan error
    var prefixMap map[string]string
    
    var names map[string]string
    var stack []stackEntry
%}

%union{
    s string
    t argo.Term
}

%token <s> A AS BNODE DT EOF IDENTIFIER IRIREF NAME STRING

%type <s> identifier qname raw_iriref slash_separated_name slashed_extension slashed_extensions
%type <t> bnode description iriref literal object predicate raw_subject subject

%%

squirtle    : statements EOF                                {return 0}

statements  : statements statement
            | statement

statement   : name_decl
            | description

name_decl   : NAME raw_iriref AS IDENTIFIER                 {names[$4] = $2; prefixMap[$2] = $4}

description : subject description_body                      {$$ = $1; stack = stack[:len(stack) - 1]}

description_body    : '{' predicate_object_list '}'

subject : raw_subject                                       {$$ = $1; stack = append(stack, stackEntry{$$, 1})}

raw_subject : iriref                                        {$$ = $1}
            | bnode                                         {$$ = $1}
            |                                               {$$ = argo.NewAnonNode()}

predicate_object_list   : predicate_object_list predicate_object    
                        | predicate_object

predicate_object    : predicate object                      {tripleChan <- argo.NewTriple(stack[len(stack) - 1].Subject, $1, $2)}

predicate   : iriref                                        {$$ = $1}
            | A                                             {$$ = argo.A}
            | '*'                                           {$$ = argo.RDF.Get(fmt.Sprintf("_%d", stack[len(stack) - 1].NextItem)); stack[len(stack) - 1].NextItem++}

object  : bnode                                             {$$ = $1}
        | literal                                           {$$ = $1}
        | iriref                                            {$$ = $1}
        | description                                       {$$ = $1}

bnode   : BNODE IDENTIFIER                                  {$$ = argo.NewBlankNode($2)}

literal : STRING                                            {$$ = argo.NewLiteral($1)}
        | STRING '@' IDENTIFIER                             {$$ = argo.NewLiteralWithLanguage($1, $3)}
        | STRING DT iriref                                  {$$ = argo.NewLiteralWithDatatype($1, $3)}

iriref  : raw_iriref                                        {$$ = argo.NewResource($1)}

raw_iriref  : IRIREF                                        {$$ = $1}
            | qname                                         {$$ = $1}
            | slash_separated_name                          {$$ = $1}
            | IDENTIFIER                                    {$$ = names[$1]}

qname   : IDENTIFIER ':' identifier                         {$$ = addHash(names[$1]) + $3}

slash_separated_name    : IDENTIFIER slashed_extensions     {$$ = stripSlash(names[$1]) + $2}

slashed_extensions  : slashed_extensions slashed_extension  {$$ = $1 + $2}
                    | slashed_extension                     {$$ = $1}

slashed_extension   : '/' identifier                        {$$ = "/" + $2}

identifier  : IDENTIFIER                                    {$$ = $1}
            | A                                             {$$ = $1}
            | AS                                            {$$ = $1}
            | NAME                                          {$$ = $1}

%%

func addHash(s string) (r string) {
    last := s[len(s) - 1]
    if last != '#' && last != '/' {
        return s + "#"
    }
    
    return s
}

func stripSlash(s string) (r string) {
    last := s[len(s) - 1]
    if last == '#' || last == '/' {
        return s[:len(s) - 1]
    }
    
    return s
}

type lexer struct {
    input *bufio.Reader
    currentToken []rune
    currentTokenLen int
    lastTokenLen int
    lastColumn int
    lineno int
    column int
}

func newLexer(input io.Reader) (ll *lexer) {
    ll = &lexer{
        input: bufio.NewReader(input),
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
        ll.AcceptRun(func(r rune) bool {return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'})
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
        
        ll.AcceptRun(func(r rune) bool {return r != '>'})
        lval.s = ll.GetToken()
        
        ll.Next()
        ll.Discard()
        
        return IRIREF
    
    case r == '"':
        ll.Discard()
        
        ll.AcceptRun(func(r rune) bool {return r != '"'})
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
    for f(ll.Next()) {}
    
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
