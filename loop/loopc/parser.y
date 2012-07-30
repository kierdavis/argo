%{
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
        "rdf": "http://www.w3.org/1999/02/22-rdf-syntax-ns#",
        "rdfs": "http://www.w3.org/2000/01/rdf-schema#",
        "owl": "http://www.w3.org/2002/07/owl#",
        "foaf": "http://xmlns.com/foaf/0.1/",
        "dc": "http://purl.org/dc/elements/1.1/",
        "dct": "http://purl.org/dc/terms/",
        "loop": "http://kierdavis.com/data/vocab/loop/",
        "xsd": "http://www.w3.org/2001/XMLSchema#",
    }
    
    var currentFuncURI string
    var currentVars map[string]string
    var parserOutput = make(chan *Func, 2)
%}

%union{
    lineno int
    column int
    name string
    
    s string
    i int64
    f float64
    A *Arg
    AL []*Arg
    T Type
    S Statement
    SL []Statement
    E Expression
    EM map[string]Expression
    PO *PredicateObjectPair
    O Object
    L *LiteralObject
}

%token <i> INTCONST
%token <f> FLOATCONST
%token <s> IDENTIFIER QUOTEDSTRING URIREF
%token <s> AS BOOLEAN DATA DTSYMBOL FALSE FLOAT FUNC INTEGER NAME OF RESOURCE STRING TRUE

%type <s> func_uri opt_label qname resource resource_or_argument resource_or_name resource_or_variable slashed_extension slashed_extensions slashed_reference
%type <A> arg
%type <AL> args opt_args_inner opt_args_outer
%type <T> atomic_type opt_type type
%type <S> stmt
%type <SL> func_body stmts
%type <E> atomic_expr expr funccall nonatomic_expr
%type <EM> namedexprlist
%type <PO> predicate_object
%type <O> object
%type <L> literal

%%

start       : toplevels
                {
                    close(parserOutput)
                }
            
            ;

toplevels   : /* empty */
            | toplevels toplevel
            ;

toplevel    : namedef
            | funcdef
            ;

namedef : NAME resource_or_name AS IDENTIFIER 
            {
                nameMap[$4] = $2
            }
        
        ;

funcdef : FUNC func_uri opt_args_outer func_body
            {
                parserOutput <- &Func{$2, $3, $4}
                currentFuncURI = ""
            }
        ;

func_uri    : resource_or_name
                {
                    $$ = $1
                    currentFuncURI = $$
                    currentVars = make(map[string]string)
                    
                    last := currentFuncURI[len(currentFuncURI) - 1]
                    if last == '/' || last == '#' {
                        currentFuncURI = currentFuncURI[:len(currentFuncURI) - 1]
                    }
                }
            
            ;

func_body   : '{' stmts '}'
                {
                    $$ = $2
                }
            
            | stmt
                {
                    $$ = []Statement{$1}
                }
            
            ;

stmts   : stmts stmt
            {
                $$ = append($1, $2)
            }
        
        | stmt
            {
                $$ = []Statement{$1}
            }
        
        ;

stmt    : predicate_object
            {
                $$ = $1
            }
        
        | nonatomic_expr
            {
                $$ = $1
            }
        
        ;

predicate_object    : resource_or_name object
                        {
                            $$ = &PredicateObjectPair{"", $1, $2}
                        }
                    
                    | resource_or_name OF IDENTIFIER object
                        {
                            $$ = &PredicateObjectPair{currentVars[$3], $1, $4}
                        }
                    
                    ;

object  : resource_or_name
            {
                $$ = ResourceObject($1)
            }
        
        | literal
            {
                $$ = $1
            }
        
        ;

literal : QUOTEDSTRING
            {
                $$ = &LiteralObject{$1, "", ""}
            }
        
        | QUOTEDSTRING '@' IDENTIFIER
            {
                $$ = &LiteralObject{$1, $3, ""}
            }
        
        | QUOTEDSTRING DTSYMBOL resource_or_name
            {
                $$ = &LiteralObject{$1, "", $3}
            }
        
        ;

expr    : nonatomic_expr
            {
                $$ = $1
            }
        
        | atomic_expr
            {
                $$ = $1
            }
        
        ;

nonatomic_expr  : funccall
                    {
                        $$ = $1
                    }
                
                ;

atomic_expr : resource_or_variable
                {
                    $$ = ResourceConstant($1)
                }
            
            | INTCONST
                {
                    $$ = IntegerConstant($1)
                }
            
            | FLOATCONST
                {
                    $$ = FloatConstant($1)
                }
            
            | TRUE
                {
                    $$ = BooleanConstant(true)
                }
            
            | FALSE
                {
                    $$ = BooleanConstant(false)
                }
            
            | literal
                {
                    lit := $1
                    
                    switch lit.Datatype {
                    case XSDboolean:
                        switch lit.Value {
                        case "true", "1":
                            $$ = BooleanConstant(true)
                        
                        case "false", "0":
                            $$ = BooleanConstant(false)
                        
                        default:
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for boolean constant: %s", 0, 0, lit.Value)
                        }
                    
                    case XSDbase64Binary:
                        data, err := base64.StdEncoding.DecodeString(lit.Value)
                        if err != nil {
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for base64 constant: %s (%s)", 0, 0, lit.Value, err.Error())
                        } else {
                            $$ = DataConstant(data)
                        }
                    
                    case XSDhexBinary:
                        data,err := hex.DecodeString(lit.Value)
                        if err != nil {
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for hex constant: %s (%s)", 0, 0, lit.Value, err.Error())
                        } else {
                            $$ = DataConstant(data)
                        }
                    
                    case XSDfloat, XSDdecimal, XSDdouble:
                        n, err := strconv.ParseFloat(lit.Value, 64)
                        if err != nil {
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for float constant: %s (%s)", 0, 0, lit.Value, err.Error())
                        } else {
                            $$ = FloatConstant(n)
                        }
                    
                    case XSDinteger, XSDnonPositiveInteger, XSDnegativeInteger, XSDlong, XSDint, XSDshort, XSDbyte, XSDnonNegativeInteger, XSDunsignedLong, XSDunsignedInt, XSDunsignedShort, XSDunsignedByte, XSDpositiveInteger:
                        n, err := strconv.ParseInt(lit.Value, 10, 64)
                        if err != nil {
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for integer constant: %s (%s)", 0, 0, lit.Value, err.Error())
                        } else {
                            $$ = IntegerConstant(n)
                        }
                    
                    case XSDanyURI:
                        $$ = ResourceConstant(lit.Value)
                    
                    case XSDQName:
                        colonPos := strings.Index(lit.Value, ":")
                        if colonPos < 0 {
                            fmt.Fprintf(os.Stderr, "[line %d column %d] Invalid value for QName constant: %s", 0, 0, lit.Value)
                        } else {
                            a := lit.Value[:colonPos]
                            b := lit.Value[colonPos+1:]
                            $$ = ResourceConstant(nameMap[a] + b)
                        }
                    
                    default:
                        $$ = StringConstant(lit.Value)
                    }
                }
            
            ;

funccall    : resource_or_name '(' namedexprlist ')'
                {
                    $$ = &FuncCall{$1, $3}
                }
            
            | resource_or_name '(' ')'
                {
                    $$ = &FuncCall{$1, map[string]Expression{}}
                }

            ;

namedexprlist   : namedexprlist ',' resource_or_name expr
                    {
                        $$ = $1
                        $$[$3] = $4
                    }
                
                | resource_or_name expr
                    {
                        $$ = map[string]Expression{$1: $2}
                    }
                
                ;

opt_args_outer  : /* empty */
                    {
                        $$ = []*Arg{}
                    }

                | '(' opt_args_inner ')'
                    {
                        $$ = $2
                    }
                
                ;

opt_args_inner  : /* empty */
                    {
                        $$ = []*Arg{}
                    }
                
                | args
                    {
                        $$ = $1
                    }
                
                ;

args    : args ',' arg
            {
                $$ = append($1, $3)
            }
        
        | arg
            {
                $$ = []*Arg{$1}
            }
        
        ;

arg : resource_or_argument opt_type opt_label
        {
            $$ = &Arg{$1, $2, $3}
        }

    ;

resource_or_argument    : IDENTIFIER
                            {
                                name := $1
                                uri, ok := nameMap[name]
                                
                                if ok {
                                    $$ = uri
                                } else {
                                    $$ = currentFuncURI + "/arg/" + name
                                    currentVars[name] = $$
                                }
                            }
                        
                        | resource
                            {
                                $$ = $1
                            }
                        
                        ;

resource_or_variable    : IDENTIFIER
                            {
                                name := $1
                                uri, ok := currentVars[name]
                                
                                if ok {
                                    $$ = uri
                                } else {
                                    $$ = nameMap[name]
                                }
                            }
                        
                        | resource
                            {
                                $$ = $1
                            }
                        
                        ;

opt_type    : /* empty */
                {
                    $$ = nil
                }
            
            | type
                {
                    $$ = $1
                }
            
            ;

type    : atomic_type
            {
                $$ = $1
            }
        
        ;

atomic_type : BOOLEAN
                {
                    $$ = Boolean
                }
            
            | FLOAT
                {
                    $$ = Float
                }
            
            | INTEGER
                {
                    $$ = Integer
                }
            
            | RESOURCE
                {
                    $$ = Resource
                }
            
            | STRING
                {
                    $$ = String
                }
            
            ;

opt_label   : /* empty */
                {
                    $$ = ""
                }
            
            | QUOTEDSTRING
                {
                    $$ = $1
                }
            
            ;

resource_or_name    : resource
                        {
                            $$ = $1
                        }
                    
                    | IDENTIFIER
                        {
                            $$ = nameMap[$1]
                        }
                    
                    ;

resource    : URIREF
                {
                    $$ = $1
                }
            
            | qname
                {
                    $$ = $1
                }
            
            | slashed_reference
                {
                    $$ = $1
                }
            
            ;

qname   : IDENTIFIER ':' IDENTIFIER
            {
                base := nameMap[$1]
                if len(base) > 0 {
                    last := base[len(base) - 1]
                    if last != '/' && last != '#' {
                        base += "#"
                    }
                
                } else {
                    base = "#"
                }
                
                $$ = base + $3
            }
        
        ;

slashed_reference   : IDENTIFIER slashed_extensions
                        {
                            base := nameMap[$1]
                            last := base[len(base) - 1]
                            if last == '/' || last == '#' {
                                base = base[:len(base) - 1]
                            }
                            
                            $$ = base + $2
                        }
                    
                    ;

slashed_extensions  : slashed_extensions slashed_extension
                        {
                            $$ = $1 + $2
                        }
                    
                    | slashed_extension
                        {
                            $$ = $1
                        }
                    
                    ;

slashed_extension   : '/' IDENTIFIER
                        {
                            $$ = "/" + $2
                        }
                    
                    ;

%%

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
