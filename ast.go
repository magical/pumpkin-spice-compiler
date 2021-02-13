package main

type Expr interface{}

type VarExpr struct {
	Name string
}

type IntExpr struct {
	Value string
}

type BinExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

type CallExpr struct {
	Func Expr
	Args []Expr
}

type DotExpr struct {
	Op    string
	Left  Expr
	Right string
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
	Name string
	Args []string
	Body Expr
}
