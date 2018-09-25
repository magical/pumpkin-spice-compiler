package main

import (
	"fmt"
	"strconv"
)

// lower.go is the middle-end of the compiler
// it takes a top-level Expr and lowers it down to a Prog

type Prog struct {
	blocks []*Block
}

// A block is the basic building-block of the low-level code.
// A block takes some arguments, executes some code, and jumps to another block.
// A function is a block whose parameters include a stack frame, a lexical closure, and a continuation.
type Block struct {
	name string
	code []Lop
}

type Reg string

type Lopcode int

const (
	Lnoop Lopcode = iota

	Linit // a = k
	Lcopy // a = b
	Ladd  // a = b + c
	Lsub  // a = b - c
	Lmul  // a = b * c
	Ldiv  // a = b / c
	Lcall // b(c)
)

// A lop is a low-level operation
type Lop struct {
	Op      Lopcode
	A, B, C Reg
	K       int
}

type compiler struct {
	funcs   []*Func
	blocks  []*Block
	code    []Lop
	dst     Reg
	lastreg int64
}

// lower generates bytecode from an expr
func lower(expr Expr) *Prog {
	c := new(compiler)
	// first pass: resolve all symbols & scopes
	// TODO
	expr = c.resolveScopes(expr)
	// second pass: extract functions
	c.extractFuncs(expr)
	// third pass: convert functions to lops
	c.genFuncs()

	return &Prog{c.blocks}
}

func (c *compiler) resolveScopes(expr Expr) Expr {
	// TODO
	return expr
}

func (c *compiler) extractFuncs(expr Expr) {
	c.funcs = nil
	c.funcs = append(c.funcs, &Func{Name: "<toplevel>", Body: expr})
	c.visitFuncs(expr)
}

func (c *compiler) visitFuncs(expr Expr) {
	switch v := expr.(type) {
	case string:
		// XXX
	case *IntExpr:
		// nothing
	case *BinExpr:
		c.visitFuncs(v.Left)
		c.visitFuncs(v.Right)
	case *CallExpr:
		c.visitFuncs(v.Func)
		for i := range v.Args {
			c.visitFuncs(v.Args[i])
		}
	case *LetExpr:
		c.visitFuncs(v.Val)
		c.visitFuncs(v.Body)
	case *IfExpr:
		c.visitFuncs(v.Cond)
		c.visitFuncs(v.Then)
		c.visitFuncs(v.Else)
	case *Func:
		c.funcs = append(c.funcs, v)
		c.visitFuncs(v.Body)
	default:
		panic(fmt.Sprintf("unhandled case in visitFuncs: %T", expr))
	}
}

func (c *compiler) genFuncs() {
	for _, f := range c.funcs {
		c.code = nil
		c.visitBody(f.Body)
		c.blocks = append(c.blocks, &Block{name: f.Name, code: c.code})
	}
}

func (c *compiler) visitBody(expr Expr) {
	switch v := expr.(type) {
	case string:
		// XXX
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: Reg("<load " + v + ">")})
	case *IntExpr:
		c.dst = c.newreg()
		val, err := strconv.Atoi(v.value)
		if err != nil {
			panic(err) // XXX
		}
		c.emit(Lop{Op: Linit, A: c.dst, K: val})
	case *LetExpr:
		c.visitBody(v.Val)
		val := c.dst
		c.dst = c.newreg()
		c.emit(Lop{Op: Lcopy, A: c.dst, B: val})
		c.visitBody(v.Body)
	case *BinExpr:
		c.visitBody(v.Left)
		left := c.dst // XXX this is ugly
		c.visitBody(v.Right)
		right := c.dst
		c.dst = c.newreg()
		switch v.Op {
		case "+":
			c.emit(Lop{Op: Ladd, A: c.dst, B: left, C: right})
		case "-":
			c.emit(Lop{Op: Lsub, A: c.dst, B: left, C: right})
		case "*":
			c.emit(Lop{Op: Lmul, A: c.dst, B: left, C: right})
		case "/":
			c.emit(Lop{Op: Ldiv, A: c.dst, B: left, C: right})
		default:
			panic(fmt.Sprintf("unknown op: %v", v.Op))
		}
	case *Func:
		// TODO: create a closure
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: "<closure>"})
	case *CallExpr:
		// TODO: augh
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: "<call>"})
	default:
		panic(fmt.Sprintf("unhandled case in visitBody: %T", expr))
	}
}

func (c *compiler) emit(l Lop) {
	c.code = append(c.code, l)
}

func (c *compiler) newreg() Reg {
	c.lastreg++
	return Reg("r" + strconv.FormatInt(c.lastreg, 10))
}
