package main

//go:generate goyacc grammar.y

import (
	"errors"
	"fmt"
	"io"
	"text/scanner"
)

const scannerMode = scanner.ScanIdents | scanner.ScanInts | scanner.SkipComments

// lexer implements yyLexer { Lex(lval *yySymType) int; Error(e string) }
type lexer struct {
	result  Expr
	scanner scanner.Scanner
	err     error
}

func (l *lexer) Init(r io.Reader) {
	l.scanner.Error = func(s *scanner.Scanner, msg string) {
		l.Error(msg)
	}
	l.scanner.Mode = scannerMode
	l.scanner.Init(r)
}

func (l *lexer) Error(e string) {
	l.err = errors.New(e)
	fmt.Println(e)
}

func (l *lexer) Lex(lval *yySymType) int {
	r := l.scanner.Scan()
	if r == scanner.Ident {
		switch token := l.scanner.TokenText(); token {
		case "let":
			return kLet
		case "in":
			return kIn
		case "if":
			return kIf
		case "then":
			return kThen
		case "else":
			return kElse
		case "func":
			return kFunc
		case "end":
			return kEnd
		default:
			lval.ident = token
			return tIdent
		}
	}
	if r == scanner.Int {
		lval.num = l.scanner.TokenText()
		return tNumber
	}
	return int(r)
}
