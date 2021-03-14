%{

package main

%}

%union {
    ident string
    num string
    args []string
    expr Expr
    exprlist []Expr
}

%type <args> args arglist0 arglist1
%type <exprlist> exprlist0 exprlist1
%type <expr> expr body func call let if
%type <num> num
%type <ident> ident

%token <ident> tIdent
%token <num> tNumber
%token kLet kIn kIf kThen kElse kFunc kEnd

%left kAnd kOr
%left '<' '>' '='

%left '+' '-'
%left '*' '/'
%left unary
%left '('       // function call
%left '.'

%%

top: expr { yylex.(*lexer).result = $1 }

expr: ident { $$ = &VarExpr{$1} }
expr: num   { $$ = &IntExpr{$1} }
expr: '(' expr ')' { $$ = $2 }

// idea for a comment form which removes an entire expression
// from the parse tree. got shift/reduce conflicts so commented out for now.
//expr: '#' '(' expr ')' expr { $$ = $5 }

// TODO: boolean operators should have mutually undefined precedence
expr: expr kAnd expr { $$ = &AndExpr{$1, $3} }
expr: expr kOr expr { $$ = &OrExpr{$1, $3} }

expr: expr '.' ident { $$ = &DotExpr{".", $1, $3} }

expr: expr '=' '=' expr %prec '=' { $$ = &BinExpr{"eq", $1, $4} }
expr: expr '<' '=' expr %prec '<' { $$ = &BinExpr{"<=", $1, $4} }
expr: expr '>' '=' expr %prec '>' { $$ = &BinExpr{">=", $1, $4} }
expr: expr '<' expr { $$ = &BinExpr{"<", $1, $3} }
expr: expr '>' expr { $$ = &BinExpr{">", $1, $3} }

expr: expr '+' expr { $$ = &BinExpr{"+", $1, $3} }
expr: expr '-' expr { $$ = &BinExpr{"-", $1, $3} }
expr: expr '*' expr { $$ = &BinExpr{"*", $1, $3} }
expr: expr '/' expr { $$ = &BinExpr{"/", $1, $3} }

expr: '-' expr %prec unary { $$ = &BinExpr{"-", &IntExpr{"0"}, $2} }

expr: let
let: kLet ident '=' expr kIn expr kEnd { $$ = &LetExpr{Var: $2, Val: $4, Body: $6} }

expr: if
if: kIf expr kThen expr kElse expr kEnd { $$ = &IfExpr{$2, $4, $6} }

expr: func
func: kFunc        '(' args ')' body kEnd { $$ = &FuncExpr{"", $3, $5} }
func: kFunc tIdent '(' args ')' body kEnd { $$ = &FuncExpr{$2, $4, $6} }
args: arglist0
body: expr

arglist0:       { $$ = nil }
arglist0: arglist1
arglist0: arglist1 ','
arglist1: ident              { $$ = []string{$1} }
arglist1: arglist1 ',' ident { $$ = append($1, $3) }

expr: call
call: expr '(' exprlist0 ')' { $$ = &CallExpr{Func: $1, Args: $3} }

exprlist0: { $$ = nil }
exprlist0: exprlist1
exprlist0: exprlist1 ','
exprlist1: expr               { $$ = []Expr{$1} }
exprlist1: exprlist1 ',' expr { $$ = append($1, $3) }

ident: tIdent
num: tNumber

