package main

type Expr interface{}

type IntExpr struct {
	value string
}

type BinExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

type CallExpr struct {
	Func         Expr
	Args         []Expr
	continuation *Continuation
}

type LetExpr struct {
	Var  string
	Val  Expr
	Body Expr
}

type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

type FuncExpr struct {
	Name  string
	Args  []string
	Body  Expr
	scope *scope
}
