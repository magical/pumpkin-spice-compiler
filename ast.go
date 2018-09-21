package main

type Expr interface{}

type BinExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

type LetExpr struct {
	Var string
	Val Expr
}

type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

type Func struct {
	Name string
	Args []string
	Body Expr
}
