//PREFIX sqtl
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
        Subject Term
        Template []*Triple
    }
    
    type sqtlTemplate struct {
        ArgNames []string
        Triples []*Triple
    }
    
    var sqtlParserMutex sync.Mutex
    
    var sqtlTripleChan chan *Triple
    var sqtlErrChan chan error
    var sqtlPrefixMap map[string]string
    
    var sqtlNames map[string]string
    var sqtlTemplates map[string]*sqtlTemplate
    var sqtlStack []sqtlStackEntry
%}

%union{
    s string
    sL []string
    t Term
    tL []Term
}

%token <s> A_KWD AS BNODE DECIMAL DOUBLE DT EOF FALSE IDENTIFIER INTEGER IRIREF IS NAME NEW STRING TEMPLATE TRUE VAR

%type <s> postfix_identifier qname raw_iriref slash_separated_name slashed_extension slashed_extensions
%type <sL> opt_template_argnames opt_template_argnames_outer template_argnames
%type <t> apply_template apply_template_subject bnode description iriref literal nonempty_subject object predicate raw_subject subject var
%type <tL> object_list opt_template_args template_args

%%

squirtle    : statements EOF                                {return 0}

statements  : statements statement
            | statement

statement   : name_decl
            | description
            | template
            | apply_template

name_decl   : NAME raw_iriref AS IDENTIFIER                 {sqtlNames[$4] = $2; sqtlPrefixMap[$2] = $4}

description : subject description_body                      {$$ = $1; sqtlStack = sqtlStack[:len(sqtlStack) - 1]}

description_body    : '{' predicate_object_list '}'

template    : template_start IDENTIFIER opt_template_argnames_outer description_body
    {
        sqtlTemplates[$2] = &sqtlTemplate{
            ArgNames: $3,
            Triples: sqtlStack[len(sqtlStack) - 1].Template,
        }
        
        sqtlStack = sqtlStack[:len(sqtlStack) - 1]
    }

template_start  : TEMPLATE                                  {sqtlStack = append(sqtlStack, sqtlStackEntry{1, nil, []*Triple{}})}

opt_template_argnames_outer : '(' opt_template_argnames ')' {$$ = $2}
                            |                               {$$ = []string{}}

opt_template_argnames   : template_argnames                 {$$ = $1}
                        |                                   {$$ = []string{}}

template_argnames   : template_argnames ',' VAR             {$$ = append($1, $3)}
                    | VAR                                   {$$ = []string{$1}}

opt_template_args   : template_args                         {$$ = $1}
                    |                                       {$$ = []Term{}}

template_args   : template_args ',' object                  {$$ = append($1, $3)}
                | object                                    {$$ = []Term{$1}}

apply_template_subject  : nonempty_subject IS               {$$ = $1}
                        |                                   {$$ = NewAnonNode()}

apply_template  : apply_template_subject NEW IDENTIFIER '(' opt_template_args ')'
    {
        $$ = $1
        templateName := $3
        args := $5
        
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
                subj = $$
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

subject : raw_subject
    {
        $$ = $1
        
        var template []*Triple
        if len(sqtlStack) > 0 {
            template = sqtlStack[len(sqtlStack) - 1].Template
        }
        
        sqtlStack = append(sqtlStack, sqtlStackEntry{1, $$, template})
    }

raw_subject : nonempty_subject                              {$$ = $1}
            |                                               {$$ = NewAnonNode()}

nonempty_subject    : iriref                                {$$ = $1}
                    | bnode                                 {$$ = $1}
                    | var                                   {$$ = $1}

predicate_object_list   : predicate_object_list predicate_object    
                        | predicate_object

predicate_object    : predicate object_list
    {
        top := sqtlStack[len(sqtlStack) - 1]
        subj := top.Subject
        pred := $1
        
        for _, obj := range $2 {
            triple := NewTriple(subj, pred, obj)
            
            if top.Template != nil {
                top.Template = append(top.Template, triple)
            } else {
                sqtlTripleChan <- triple
            }
        }
        
        sqtlStack[len(sqtlStack) - 1] = top
    }

predicate   : iriref                                        {$$ = $1}
            | var                                           {$$ = $1}
            | A_KWD                                         {$$ = A}
            | '*'                                           {$$ = RDF.Get(fmt.Sprintf("_%d", sqtlStack[len(sqtlStack) - 1].NextItem)); sqtlStack[len(sqtlStack) - 1].NextItem++}

object_list : object_list ',' object                        {$$ = append($1, $3)}
            | object                                        {$$ = []Term{$1}}

object  : iriref                                            {$$ = $1}
        | bnode                                             {$$ = $1}
        | literal                                           {$$ = $1}
        | var                                               {$$ = $1}
        | description                                       {$$ = $1}
        | apply_template                                    {$$ = $1}

bnode   : BNODE IDENTIFIER                                  {$$ = NewBlankNode($2)}

literal : STRING                                            {$$ = NewLiteral($1)}
        | STRING '@' IDENTIFIER                             {$$ = NewLiteralWithLanguage($1, $3)}
        | STRING DT iriref                                  {$$ = NewLiteralWithDatatype($1, $3)}
        | INTEGER                                           {$$ = NewLiteralWithDatatype($1, XSD.Get("integer"))}
        | DECIMAL                                           {$$ = NewLiteralWithDatatype($1, XSD.Get("decimal"))}
        | DOUBLE                                            {$$ = NewLiteralWithDatatype($1, XSD.Get("double"))}
        | TRUE                                              {$$ = NewLiteralWithDatatype("true", XSD.Get("boolean"))}
        | FALSE                                             {$$ = NewLiteralWithDatatype("false", XSD.Get("boolean"))}

var : VAR                                                   {$$ = &sqtlVar{$1}}

iriref  : raw_iriref                                        {$$ = NewResource($1)}

raw_iriref  : IRIREF                                        {$$ = $1}
            | qname                                         {$$ = $1}
            | slash_separated_name                          {$$ = $1}
            | IDENTIFIER                                    {$$ = getName($1)}

qname   : IDENTIFIER ':' postfix_identifier                 {$$ = addHash(getName($1)) + $3}

slash_separated_name    : IDENTIFIER slashed_extensions     {$$ = stripSlash(getName($1)) + $2}

slashed_extensions  : slashed_extensions slashed_extension  {$$ = $1 + $2}
                    | slashed_extension                     {$$ = $1}

slashed_extension   : '/' postfix_identifier                {$$ = "/" + $2}

postfix_identifier  : IDENTIFIER                            {$$ = $1}
            | A_KWD                                             {$$ = $1}
            | AS                                            {$$ = $1}
            | FALSE                                         {$$ = $1}
            | IS
            | NAME                                          {$$ = $1}
            | TEMPLATE
            | TRUE                                          {$$ = $1}

%%

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
    
    last := s[len(s) - 1]
    if last != '#' && last != '/' {
        return s + "#"
    }
    
    return s
}

func stripSlash(s string) (r string) {
    if s == "" {
        return ""
    }
    
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
        ll.AcceptRun(func(r rune) bool {return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'})
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
        
        digitFunc := func(r rune) bool {return '0' <= r && r <= '9'}
        
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
        for ll.Next() != '\n' {}
        
        return ll.Lex(lval)
    
    case r == '?', r == '$':
        ll.Discard()
        ll.AcceptRun(func(r rune) bool {return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'})
        lval.s = ll.GetToken()
        
        return VAR
    
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
