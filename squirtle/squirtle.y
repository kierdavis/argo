%{
    package main
    
    import (
        "bufio"
        "fmt"
        "github.com/kierdavis/argo"
        "io"
        "os"
        "strings"
        "unicode"
        "unicode/utf8"
    )
    
    type StackEntry struct {
        Subject argo.Term
        NextItem int
    }
    
    var graph = argo.NewGraph(argo.NewListStore())
    var names = make(map[string]string)
    var stack = make([]StackEntry, 0)
%}

%union{
    s string
    t argo.Term
}

%token <s> A AS BNODE DT EOF IDENTIFIER IRIREF NAME STRING

%type <s> qname raw_iriref slash_separated_name slashed_extension slashed_extensions
%type <t> bnode description iriref literal object predicate subject

%%

squirtle    : statements EOF                                {return 0}

statements  : statements statement
            | statement

statement   : name_decl
            | description

name_decl   : NAME raw_iriref AS IDENTIFIER                 {names[$4] = $2; graph.Bind($2, $4)}

description : subject description_body                      {$$ = $1; stack = stack[:len(stack) - 1]}

description_body    : '{' predicate_object_list '}'

subject : raw_subject                                       {$$ = $1; stack = append(stack, StackEntry{$$, 1})}

raw_subject : iriref                                        {$$ = $1}
            | bnode                                         {$$ = $1}
            |                                               {$$ = argo.NewAnonNode()}

predicate_object_list   : predicate_object_list predicate_object    
                        | predicate_object

predicate_object    : predicate object                      {graph.AddTriple(stack[len(stack) - 1].Subject, $1, $2)}

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

qname   : IDENTIFIER ':' IDENTIFIER                         {$$ = addHash(names[$1]) + $3}

slash_separated_name    : IDENTIFIER slashed_extensions     {$$ = stripSlash($1) + $2}

slashed_extensions  : slashed_extensions slashed_extension  {$$ = $1 + $2}
                    | slashed_extension                     {$$ = $1}

slashed_extension   : '/' IDENTIFIER                        {$$ = "/" + $2}

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

type Lexer struct {
    input *bufio.Reader
    currentToken []rune
    currentTokenLen int
    lastTokenLen int
    lineno int
    column int
}

func NewLexer(input io.Reader) (lexer *Lexer) {
    lexer = &Lexer{
        input: bufio.NewReader(input),
        lineno: 1,
        column: 1,
    }
    
    return lexer
}

func (lexer *Lexer) Error(s string) {
    fmt.Fprintf(os.Stderr, "Syntax error: %s (at line %d col %d)\n", s, lexer.lineno, lexer.column)
    panic("Exiting due to error")
}

func (lexer *Lexer) Lex(lval *yySymType) (t int) {
    lexer.AcceptRun(unicode.IsSpace)
    lexer.Discard()
    
    r := lexer.Next()
    
    switch {
    case r == '_':
        if lexer.Accept(':') {
            lexer.Discard()
            return BNODE
        }
        
        fallthrough
    
    case unicode.IsLetter(r):
        lexer.AcceptRun(func(r rune) bool {return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'})
        lval.s = lexer.GetToken()
        
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
        lexer.Discard()
        
        lexer.AcceptRun(func(r rune) bool {return r != '>'})
        lval.s = lexer.GetToken()
        
        lexer.Next()
        lexer.Discard()
        
        return IRIREF
    
    case r == '"':
        lexer.Discard()
        
        lexer.AcceptRun(func(r rune) bool {return r != '"'})
        lval.s = lexer.GetToken()
        
        lexer.Next()
        lexer.Discard()
        
        return STRING
    
    case r == '^':
        if lexer.Accept('^') {
            lexer.Discard()
            return DT
        }
        
        lexer.Discard()
        return '^'
    }
    
    lexer.Discard()
    return int(r)
}

func (lexer *Lexer) Next() (r rune) {
    r, n, err := lexer.input.ReadRune()
    if err != nil {
        if err == io.EOF {
            return EOF
        }
        
        lexer.Error(err.Error())
    }
    
    lexer.currentToken = append(lexer.currentToken, r)
    lexer.currentTokenLen += n
    lexer.lastTokenLen = n
    
    if r == '\n' {
        lexer.lineno++
        lexer.column = 1
    } else {
        lexer.column++
    }
    
    return r
}

func (lexer *Lexer) Back() {
    err := lexer.input.UnreadRune()
    if err == nil {
        lexer.currentToken = lexer.currentToken[:len(lexer.currentToken)-1]
        lexer.currentTokenLen -= lexer.lastTokenLen
        lexer.column--
    }
}

func (lexer *Lexer) Peek() (r rune) {
    r = lexer.Next()
    lexer.Back()
    return r
}

func (lexer *Lexer) Accept(r rune) (ok bool) {
    if lexer.Next() == r {
        return true
    }
    
    lexer.Back()
    return false
}

func (lexer *Lexer) AcceptRun(f func(rune) bool) {
    for f(lexer.Next()) {}
    
    lexer.Back()
}

func (lexer *Lexer) GetToken() (s string) {
    buf := make([]byte, lexer.currentTokenLen)
    pos := 0
    
    for _, r := range lexer.currentToken {
        pos += utf8.EncodeRune(buf[pos:], r)
    }
    
    lexer.Discard()
    return string(buf)
}

func (lexer *Lexer) Discard() {
    lexer.currentToken = nil
    lexer.currentTokenLen = 0
}

func main() {
    f, err := os.Open(os.Args[1])
    if err != nil {
        panic(err)
    }
    defer f.Close()
    
    yyParse(NewLexer(f))
    
    graph.Serialize(argo.SerializeNTriples, os.Stdout)
}
