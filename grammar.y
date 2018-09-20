%{

package main

%}

%union {
    ident string
    num string
}

%token tIdent tNumber
%token kLet kIn kIf kThen kElse kFunc kEnd

%left '+' '-'
%left '*' '/'
%left '('

%%

top: expr

expr: ident
expr: num
expr: '(' expr ')'

expr: expr '+' expr
expr: expr '-' expr
expr: expr '*' expr
expr: expr '/' expr

expr: let
let: kLet ident '=' expr kIn expr kEnd

expr: if
if: kIf expr kThen expr kElse expr kEnd

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

