package main

type Expr interface{}

type VarExpr struct {
	Name string
}

type BoolExpr struct {
	Value bool
}

type IntExpr struct {
	Value string
}

type BinExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

type AndExpr struct {
	Left  Expr
	Right Expr
}

type OrExpr struct {
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

// these are used internally by the compiler
// they are not created during parsing

type TupleExpr struct {
	Args []Expr
}

type TupleIndexExpr struct {
	Base  Expr
	Index int
}
