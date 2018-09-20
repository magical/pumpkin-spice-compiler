%{

package main

%}

%union {
    ident string
    num string
}

%token tIdent tNumber
%token kLet kIf kElse kFunc kIn kEnd

%left '+' '-'
%left '*' '/'
%left '('

%%

top: expr

expr: let
let: kLet ident '=' expr kIn expr kEnd

expr: expr '+' expr
expr: expr '-' expr
expr: expr '*' expr
expr: expr '/' expr

expr: ident
expr: num
expr: '(' expr ')'

expr: func
func: kFunc '(' args ')' body kEnd
args: arglist0
body: expr

arglist0:
arglist0: arglist1
arglist0: arglist1 ','
arglist1: ident
arglist1: arglist1 ',' ident

expr: call
call: expr '(' exprlist0 ')'

exprlist0:
exprlist0: exprlist1
exprlist0: exprlist1 ','
exprlist1: expr
exprlist1: exprlist1 ',' expr

ident: tIdent
num: tNumber

