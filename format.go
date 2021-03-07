package main

import (
	"bytes"
	"fmt"
	"os"
)

// format.go converts an AST back to source code

type formatter struct {
	buf     bytes.Buffer
	nindent int
}

func printExpr(expr Expr) {
	var f formatter
	f.visitExpr(expr, 0)
	f.write("\n")
	f.buf.WriteTo(os.Stdout)
}

var binOpPrec = map[string]int{
	"and": 1,
	"or":  1,
	"eq":  2,
	"<":   2,
	"<=":  2,
	"=>":  2,
	">":   2,
	"+":   3,
	"-":   3,
	"*":   4,
	"/":   4,
	".":   5,
}

func (f *formatter) visitExpr(e Expr, prec int) {
	switch e := e.(type) {
	case *VarExpr:
		f.write(e.Name)
	case *IntExpr:
		f.write(e.Value)
	case *BoolExpr:
		if e.Value == true {
			f.write("#true")
		} else {
			f.write("#false")
		}
	case *BinExpr:
		op := binOpPrec[e.Op]
		if op < prec {
			f.write("(")
		}
		f.visitExpr(e.Left, op)
		f.write(" " + e.Op + " ")
		f.visitExpr(e.Right, op)
		if op < prec {
			f.write(")")
		}
	case *AndExpr:
		op := binOpPrec["and"]
		if op < prec {
			f.write("(")
		}
		f.visitExpr(e.Left, op)
		f.write(" and ")
		f.visitExpr(e.Right, op)
		if op < prec {
			f.write(")")
		}
	case *OrExpr:
		op := binOpPrec["or"]
		if op < prec {
			f.write("(")
		}
		f.visitExpr(e.Left, op)
		f.write(" or ")
		f.visitExpr(e.Right, op)
		if op < prec {
			f.write(")")
		}
	case *CallExpr:
		f.visitExpr(e.Func, 0)
		f.write("(")
		for i, a := range e.Args {
			if i != 0 {
				f.write(", ")
			}
			f.visitExpr(a, 0)
		}
		f.write(")")
	case *DotExpr:
		f.visitExpr(e.Left, binOpPrec["."])
		f.write(".")
		//f.visitExpr(e.Right, 0)
		f.write(e.Right)
	case *LetExpr:
		f.write("let " + e.Var + " = ")
		f.visitExpr(e.Val, 0)
		f.write(" in")
		f.indent()
		f.visitExpr(e.Body, 0)
		f.dedent()
		f.write("end")
	case *IfExpr:
		f.write("if ")
		f.visitExpr(e.Cond, 0)
		f.write(" then")
		f.indent()
		f.visitExpr(e.Then, 0)
		if e.Else != nil {
			f.dedent()
			f.write("else")
			f.indent()
			f.visitExpr(e.Else, 0)
		}
		f.dedent()
		f.write("end")
	case *FuncExpr:
		f.write("func " + e.Name + "(")
		for i, name := range e.Args {
			if i != 0 {
				f.write(", ")
			}
			f.write(name)
		}
		f.write(")")
		f.indent()
		f.visitExpr(e.Body, 0)
		f.dedent()
		f.write("end")
	case *TupleExpr:
		f.write("#tuple(")
		for i, a := range e.Args {
			if i != 0 {
				f.write(", ")
			}
			f.visitExpr(a, 0)
		}
		f.write(")")
	case *TupleIndexExpr:
		f.write("#get(")
		f.visitExpr(e.Base, 0)
		f.write(", ")
		f.write(fmt.Sprint(e.Index))
		f.write(")")
	default:
		panic(fmt.Sprintf("unhandled case in formatter.visitExpr: %T", e))
	}
}

func (f *formatter) indent() {
	f.nindent++
	f.write("\n")
	for i := 0; i < f.nindent; i++ {
		f.write("  ")
	}
}

func (f *formatter) dedent() {
	f.nindent--
	f.write("\n")
	for i := 0; i < f.nindent; i++ {
		f.write("  ")
	}
}

func (f *formatter) write(s string) {
	f.buf.WriteString(s)
}
