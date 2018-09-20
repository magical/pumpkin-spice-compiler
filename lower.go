package main

// lower.go is the middle-end of the compiler
// it takes a top-level Expr and lowers it down to a Prog

type Prog struct {
	funcs []*Func
	expr  Expr
}
