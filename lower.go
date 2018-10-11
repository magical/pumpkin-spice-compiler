package main

import (
	"fmt"
	"strconv"
)

// lower.go is the middle-end of the compiler
// it takes a top-level Expr and lowers it down to a Prog
//
// it needs a new name.

type Prog struct {
	blocks []*block
}

type Func struct {
	Name  string
	Entry *block
}

// A block is the basic building-block of the low-level code.
// A block takes some arguments, executes some code, and jumps to another block.
// A function is a block whose parameters include a stack frame, a lexical closure, and a continuation.
type block struct {
	name  string
	Func  *Func
	scope *scope
	args  []Reg
	code  []Op
}

type Reg string
type Opcode int

const (
	Noop Opcode = iota // noop

	BinOp // %a = binop "+" %x %y
	//ArithOp       // %a = arith "+" %x %y
	//CompareOp     // %a = compare "==" %x %y

	BranchOp      // branch %c -> label j, label k
	JumpOp        // jump label a(%x, %y, %z)
	CallOp        // %a, %b, ... = call %f, %x, %y, ...
	ReturnOp      // return %a, %b, ...
	LiteralOp     // %a = literal <value>
	FuncLiteralOp // %a = function_literal <function_name>

	AllocOp // %m = alloc
	FreeOp  // free %m
	LoadOp  // %a = load %m
	StoreOp // store %m, %x

	//CallWithContinuationOp //  tailcall %f, %x, %y -> label a
)

type Label string //???

type Op struct {
	Opcode  Opcode
	Variant string
	Dst     []Reg
	Src     []Reg
	Label   []Label
	Value   interface{} // for LiteralOp
	// type information?
}

type compiler struct {
	funcs   []*Func
	blocks  []*block
	lastreg int64
	errors  []error
}

type visitor struct {
	errors []error
}

// lower generates bytecode from an expr
func lower(expr Expr) *Prog {
	c := new(compiler)
	// TODO type checking??
	// first pass: resolve scopes, extract functions,
	//   and convert the AST into high-level SSA
	f := new(Func)
	b := newblock(f, "entry")
	f.Entry = b
	var s scope
	c.visitExpr(&s, b, expr)

	// second pass: CPS covert??
	//
	c.cpsConvert(expr)

	// third pass: convert functions to lops
	//

	return &Prog{} // XXX
}

type scope struct {
	vars   map[string]Expr
	parent *scope
}

func newscope(parent *scope) *scope {
	return &scope{
		vars:   make(map[string]Expr),
		parent: parent,
	}
}

func (s *scope) push() *scope {
	return newscope(s)
}

func (s *scope) define(name string) {
	if _, alreadyDefined := s.vars[name]; alreadyDefined {
		// error
	}
	s.vars[name] = 1 // XXX
}

func (s *scope) has(name string) bool { return s.lookup(name) != nil }

func (s *scope) lookup(name string) Expr {
	for s := s; s != nil; s = s.parent {
		if x, ok := s.vars[name]; ok {
			return x
		}
	}
	return nil
}

/*
func (c *compiler) visitBody(expr Expr) {
	switch v := expr.(type) {
	case string:
		// XXX
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: Reg("<load " + v + ">")})
	case *IntExpr:
		c.dst = c.newreg()
		val, err := strconv.Atoi(v.Value)
		if err != nil {
			panic(err) // XXX
		}
		c.emit(Lop{Op: Linit, A: c.dst, K: val})
	case *IfExpr:
		c.visitBody(v.Cond)
		c.visitBody(v.Then)
		c.visitBody(v.Else)
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
	case *FuncExpr:
		// TODO: create a closure
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: "<closure>"})
		// - allocate a closure object with space for
		//   the function pointer and each closed-over variable
		// - copy variables (pointers?) into closure
		// - assign closure to dst

		// a continuation is like a closure, except it reuses its containing function's stack frame(?)
	case *CallExpr:
		// TODO: augh
		// need to convert a call into continuation-passing style here
		//
		c.dst = c.newreg()
		c.emit(Lop{Op: Lnoop, A: c.dst, B: "<call>"})
		// - extract function pointer from closure
		// - create a stack frame for the function, for arguments and locals
		// - push args into stack frame
		// - jump to function, passing along the closure, stack, and continuation
	default:
		panic(fmt.Sprintf("unhandled case in visitBody: %T", expr))
	}
}
*/

func (c *compiler) errorf(format string, v ...interface{}) {
	c.errors = append(c.errors, fmt.Errorf(format, v...))
}

func (b *block) emit(l Op) {
	b.code = append(b.code, l)
}

func (c *compiler) newreg() Reg {
	c.lastreg++
	return Reg("r" + strconv.FormatInt(c.lastreg, 10))
}

func (c *compiler) newreg1() []Reg {
	return []Reg{c.newreg()}
}

func newblock(f *Func, name string) *block {
	return &block{
		name: name,
		Func: f,
	}
}

//type visitor = compiler

func (v *compiler) visitExpr(s *scope, b *block, e Expr) (dst []Reg) {
	switch e := e.(type) {
	case *VarExpr:
		if !s.has(e.Name) {
			v.errorf("%v is not in scope", e.Name)
		}
		// ???
		//ref := s.lookup(e.Name)
		ref := Reg(e.Name) // XXX
		// emit load
		dst = v.newreg1()
		b.emit(Op{
			Opcode: LoadOp,
			Dst:    dst,
			Src:    []Reg{ref},
		})
	case *IntExpr:
		// emit literal
		dst = v.newreg1()
		b.emit(Op{
			Opcode: LiteralOp,
			Value:  e.Value,
		})
	case *LetExpr:
		inner := s.push()
		v.visitExpr(s, b, e.Val)
		inner.define(e.Var)
		v.visitExpr(inner, b, e.Body)
	case *IfExpr:
		v.visitExpr(s, b, e.Cond)
		then := newblock(b.Func, "then")
		els := newblock(b.Func, "else")
		v.visitExpr(s, then, e.Then)
		v.visitExpr(s, els, e.Else)
		v.blocks = append(v.blocks, then, els)
		// new block for afterwards?
		// emit branch...
	case *FuncExpr:
		// XXX factor this into a different function
		f := new(Func)
		entry := newblock(f, "entry")
		f.Entry = entry
		inner := s.push()
		for _, a := range e.Args {
			inner.define(a)
		}
		v.visitExpr(s, entry, e.Body)
		// TODO: emit return at end of func
		v.funcs = append(v.funcs, f)

		// also emit a function reference...
		dst = v.newreg1()
		b.emit(Op{
			Opcode: FuncLiteralOp,
			Dst:    dst,
			Src:    []Reg{Reg(f.Name)}, //???
		})

	case *CallExpr:
		f := v.visitExpr(s, b, e.Func)
		src := make([]Reg, len(e.Args)+1)
		src[0] = f[0] // XXX
		for i, a := range e.Args {
			src[i+1] = v.visitExpr(s, b, a)[0] // XXX
		}
		dst = v.newreg1()
		b.emit(Op{
			Opcode: CallOp,
			Dst:    dst,
			Src:    src,
		})
		// emit call
	case *BinExpr:
		y := v.visitExpr(s, b, e.Left)[0]
		z := v.visitExpr(s, b, e.Right)[0]
		dst = v.newreg1()
		b.emit(Op{
			Opcode: BinOp,
			Dst:    dst,
			Src:    []Reg{y, z},
		})
	default:
		panic(fmt.Sprintf("unhandled case in visitBody: %T", e))
	}
	return dst
}

func (c *compiler) cpsConvert(e Expr) { /* magic happens here */ }
