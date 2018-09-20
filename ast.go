package main

type Expr interface{}

type LetExpr struct {
	Var string
	Val Expr
}

type Func struct {
	Name string
	Args []string
	Body Expr
}

type BinExpr struct {
	Op    string
	Left  Expr
	Right Expr
}
