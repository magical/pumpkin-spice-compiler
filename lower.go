package main

import (
	"fmt"
	"os"
	"strconv"
)

// lower.go is the middle-end of the compiler
// it takes a top-level Expr and lowers it down to a Prog
//
// it needs a new name.

type Prog struct {
	funcs  []*Func
	blocks []*block
}

type Func struct {
	Name   string
	Entry  *block
	blocks []*block
}

// A block is the basic building-block of the low-level code.
// A block takes some arguments, executes some code, and jumps to another block.
// A function is a block whose parameters include a stack frame, a lexical closure, and a continuation.
type block struct {
	name  Label
	Func  *Func
	scope *scope
	args  []Reg
	code  []Op
}

type Reg string   // XXX should also have type information
type Label string //???
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

	CallWithContinuationOp //  tailcall %f, %x, %y -> label a
)

func (l Opcode) String() string {
	switch l {
	case Noop:
		return "noop"
	case BinOp:
		return "binop"
	case BranchOp:
		return "branch"
	case JumpOp:
		return "jump"
	case CallOp:
		return "call"
	case ReturnOp:
		return "return"
	case LiteralOp:
		return "literal"
	case FuncLiteralOp:
		return "function_literal"
	case AllocOp:
		return "alloc"
	case FreeOp:
		return "free"
	case LoadOp:
		return "load"
	case StoreOp:
		return "store"
	case CallWithContinuationOp:
		return "call_with_continuation"
	default:
		return fmt.Sprintf("op(%d)", int(l))
	}
}

func (l Opcode) GoString() string {
	switch l {
	case Noop:
		return "Noop"

	case BinOp:
		return "BinOp"

	case BranchOp:
		return "BranchOp"
	case JumpOp:
		return "JumpOp"
	case CallOp:
		return "CallOp"
	case ReturnOp:
		return "ReturnOp"
	case LiteralOp:
		return "LiteralOp"
	case FuncLiteralOp:
		return "FuncLiteralOp"

	case AllocOp:
		return "AllocOp"
	case FreeOp:
		return "FreeOp "
	case LoadOp:
		return "LoadOp "
	case StoreOp:
		return "StoreOp"
	default:
		return fmt.Sprintf("Opcode(%d)", int(l))
	}
}

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

func (f *Func) entry() *block {
	return f.blocks[0]
}

// lower generates bytecode from an expr
func lower(expr Expr) *Prog {
	c := new(compiler)
	// TODO type checking??
	// first pass: resolve scopes, extract functions,
	//   and convert the AST into high-level SSA
	f := new(Func)
	f.Name = "<toplevel>"
	b := newblock(f, "entry")
	var s scope
	c.funcs = append(c.funcs, f)
	exitb, val := c.visitExpr(&s, b, expr)
	// return the final value
	exitb.emit(Op{
		Opcode: ReturnOp,
		Src:    val,
	})

	// second pass: CPS covert??
	//
	c.cpsConvert()

	// third pass: lower everything to machine types?
	//

	return &Prog{funcs: c.funcs} // XXX
}

type scope struct {
	vars   map[string]interface{}
	parent *scope
}

func newscope(parent *scope) *scope {
	return &scope{
		vars:   make(map[string]interface{}),
		parent: parent,
	}
}

func (s *scope) push() *scope {
	return newscope(s)
}

// TODO: what is this? needs a new name
type mvar struct {
	Reg  Reg
	Func *Func
}

func (s *scope) define(name string) *mvar {
	if _, alreadyDefined := s.vars[name]; alreadyDefined {
		// error
	}
	v := new(mvar)
	s.vars[name] = v
	v.Reg = "<uninitialized>"
	return v
}

func (s *scope) has(name string) bool { return s.lookup(name) != nil }

func (s *scope) lookup(name string) interface{} {
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
	fmt.Fprintln(os.Stderr, c.errors[len(c.errors)-1]) // XXX
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
	// XXX uniquify name
	b := &block{
		name: Label(name),
		Func: f,
	}
	f.blocks = append(f.blocks, b)
	return b
}

//type visitor = compiler

func (v *compiler) visitExpr(s *scope, b *block, e Expr) (bl *block, dst []Reg) {
	switch e := e.(type) {
	case *VarExpr:
		if !s.has(e.Name) {
			v.errorf("%v is not in scope", e.Name)
			dst = v.newreg1() // invent a register so we don't crash
			break
		}
		// ???
		ref := s.lookup(e.Name).(*mvar)
		dst = []Reg{ref.Reg}
		/*
			// emit load
			dst = v.newreg1()
			if ref.Func != nil {
				// reference to a static function
				b.emit(Op{
					//Opcode: LoadGlobalOp,
					Dst:   dst,
					Value: ref.Func.Name,
				})
			} else {
				// reference to a variable
				// XXX if the var is from an outer scope
				// then we need to go through a closure.
				// does that mean we have to do closure conversion
				// in an earlier pass?
				b.emit(Op{
					Opcode: LoadOp,
					Dst:    dst,
					Src:    []Reg{ref.Reg},
				})
			}
		*/
	case *IntExpr:
		// emit literal
		dst = v.newreg1()
		b.emit(Op{
			Opcode: LiteralOp,
			Dst:    dst,
			Value:  e.Value,
		})
	case *LetExpr:
		/*
			// evaluate the rvalue
			var val []Reg
			b, val = v.visitExpr(s, b, e.Val)
			// allocate space for the lvalue
			m := v.newreg()
			b.emit(Op{
				Opcode: AllocOp,
				Dst:    []Reg{m},
				// Comment: e.Var,
			})
			// create a new scope
			// and add the variable to it
			inner := s.push()
			varInfo := inner.define(e.Var)
			// and store the rvalue
			varInfo.Reg = m
			b.emit(Op{
				Opcode: StoreOp,
				Src:    []Reg{m, val[0]},
			})
			// evaluate the body of the let expression
			// in the new scope
			b, dst = v.visitExpr(inner, b, e.Body)
			// finally, free the variable
			b.emit(Op{
				Opcode: FreeOp,
				Src:    []Reg{m},
			})
		*/

		// evaluate the rvalue
		var val []Reg
		b, val = v.visitExpr(s, b, e.Val)
		// create a new scope
		// and add the variable to it
		inner := s.push()
		varInfo := inner.define(e.Var)
		varInfo.Reg = val[0]
		// evaluate the body of the let expression
		// in the new scope
		b, dst = v.visitExpr(inner, b, e.Body)
	case *IfExpr:
		var cond []Reg
		b, cond = v.visitExpr(s, b, e.Cond)
		then := newblock(b.Func, "then")
		els := newblock(b.Func, "else")
		v.blocks = append(v.blocks, then, els)
		// emit branch...
		b.emit(Op{
			Opcode: BranchOp,
			Src:    cond,
			Label:  []Label{then.name, els.name},
		})

		// Join the branches
		be := newblock(b.Func, "end")
		dst = v.newreg1()
		b = be
		bt, dt := v.visitExpr(s, then, e.Then)
		bf, df := v.visitExpr(s, els, e.Else)
		bt.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			// Args: dt
		})
		bf.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			// Args: df,
		})
		_, _ = dt, df
	case *FuncExpr:
		f := v.visitFunc(s, b, e)
		// emit a function reference
		dst = v.newreg1()
		b.emit(Op{
			Opcode: FuncLiteralOp,
			Dst:    dst,
			Value:  f.Name,
		})
	case *CallExpr:
		// evaluate the function
		var tmp []Reg
		b, tmp = v.visitExpr(s, b, e.Func)
		src := make([]Reg, len(e.Args)+1)
		src[0] = tmp[0] // XXX
		// evaluate the arguments
		for i, a := range e.Args {
			b, tmp = v.visitExpr(s, b, a)
			src[i+1] = tmp[0] // XXX
		}
		// call the function
		dst = v.newreg1()
		b.emit(Op{
			Opcode: CallOp,
			Dst:    dst,
			Src:    src,
		})
	case *BinExpr:
		b1, y := v.visitExpr(s, b, e.Left)
		b2, z := v.visitExpr(s, b1, e.Right)
		b = b2
		dst = v.newreg1()
		b.emit(Op{
			Opcode:  BinOp,
			Variant: e.Op,
			Dst:     dst,
			Src:     []Reg{y[0], z[0]}, //XXX
		})
	case *DotExpr:
		var tmp []Reg
		b, tmp = v.visitExpr(s, b, e.Left)
		dst = v.newreg1()
		b.emit(Op{
			Opcode:  BinOp,
			Variant: ".",
			Dst:     dst,
			Src:     []Reg{tmp[0]},
			Value:   e.Right, //XXX
		})
	default:
		panic(fmt.Sprintf("unhandled case in visitBody: %T", e))
	}
	return b, dst
}

func (c *compiler) visitFunc(s *scope, b *block, e *FuncExpr) *Func {
	f := new(Func)
	if e.Name == "" {
		f.Name = "<lambda>"
	} else {
		f.Name = e.Name
	}
	entry := newblock(f, "entry")
	inner := s.push()
	if e.Name != "" {
		mf := inner.define(e.Name)
		mf.Func = f
	}
	for _, a := range e.Args {
		m := inner.define(a)
		m.Reg = c.newreg()
	}
	b, dst := c.visitExpr(inner, entry, e.Body)
	// return the last thing evaluated
	b.emit(Op{
		Opcode: ReturnOp,
		Src:    dst,
	})

	c.funcs = append(c.funcs, f)
	return f
}

func (c *compiler) cpsConvert() {

	/* magic happens here */

	for _, f := range c.funcs {
		// have to use an old-style loop so we pick up
		// modifications to the slice
		for jjj := 0; jjj < len(f.blocks); jjj++ {
			b := f.blocks[jjj]
			for i, l := range b.code {
				if l.Opcode == CallOp {
					fmt.Println("before")
					printb(b)

					k := newblock(f, "continuation"+fmt.Sprint(jjj))
					k.code = append([]Op(nil), b.code[i+1:]...)
					b.code = b.code[:i+1]
					b.code[i] = Op{
						Opcode: CallWithContinuationOp,
						Src:    l.Src,
						Label:  []Label{k.name},
						// LabelArgs: hmm,
					}

					fmt.Println("after")
					printb(b)
					fmt.Println("\t---")
					printb(k)

					// XXX insert k after current block instead of at end?

					// the rest of the code will be picked up when
					// the outer loop gets to the new block
					break
				}
			}
		}
	}
}
