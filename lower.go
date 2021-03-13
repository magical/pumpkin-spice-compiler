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
	Name     string
	Entry    *block
	blocks   []*block
	literals map[Reg]int64 // used during ir->asm lowering
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
	pred  []*block // predecessor blocks
	succ  []*block // successor blocks
}

type Reg string   // XXX should also have type information
type Label string //???
type Opcode int

const (
	Noop Opcode = iota // noop

	BinOp // %a = binop "+" %x %y
	//ArithOp   // %a = arith "+" %x %y
	CompareOp // %a = compare "==" %x %y

	BranchOp      // branch %c -> label j, label k
	JumpOp        // jump label a(%x, %y, %z)
	CallOp        // %a, %b, ... = call %f, %x, %y, ...
	ReturnOp      // return %a, %b, ...
	LiteralOp     // %a = literal <value>
	FuncLiteralOp // %a = function_literal <function_name>

	RecordGetOp // %a = record_get %tuple <0>
	RecordSetOp // record_set %tuple, %x <0>

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
	case CompareOp:
		return "compare"
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
	case RecordGetOp:
		return "record_get"
	case RecordSetOp:
		return "record_set"
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
	case CompareOp:
		return "CompareOp"

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
	funcs []*Func
	//blocks  []*block
	lastreg int64
	lastlab int64
	errors  []error
}

type visitor struct {
	errors []error
}

func (f *Func) entry() *block {
	if len(f.blocks) == 0 {
		b := newblock(f, "entry")
		f.blocks = append(f.blocks, b)
		return b
	}
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
	var s scope
	c.funcs = append(c.funcs, f)
	exitb, val := c.visitExpr(&s, f.entry(), expr)
	// return the final value
	exitb.emit(Op{
		Opcode: ReturnOp,
		Src:    val,
	})

	// second pass: CPS covert??
	//
	//c.cpsConvert()

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

func (c *compiler) newlabel(name string) string {
	c.lastlab++
	return name + "." + strconv.FormatInt(c.lastlab, 10)
}

func newblock(f *Func, name string) *block {
	b := &block{
		name: Label(name),
		Func: f,
	}
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
	case *BoolExpr:
		var value int64 = 0
		if e.Value {
			value = 1
		}
		dst = v.newreg1()
		b.emit(Op{
			Opcode: LiteralOp,
			Dst:    dst,
			Value:  value,
		})
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
		// Evaluate the condition
		bThen, bElse := v.visitCond(s, b, e.Cond)
		// Evaluate the branches
		bt, dt := v.visitExpr(s, bThen, e.Then)
		bf, df := v.visitExpr(s, bElse, e.Else)
		// Join the branches
		//be := b.join("end", bt, bf)
		be := newblock(b.Func, v.newlabel("end"))
		be.pred = append(be.pred, bt, bf)
		bt.succ = append(bt.succ, be)
		bf.succ = append(bf.succ, be)
		b.Func.blocks = append(b.Func.blocks, be)
		be.args = v.newreg1() // TODO: len(dt)?
		bt.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			Src:    dt,
		})
		bf.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			Src:    df,
		})
		b, dst = be, be.args
	case *AndExpr, *OrExpr:
		bThen, bElse := v.visitCond(s, b, e)
		// Evaluate the branches
		bt, dt := v.visitExpr(s, bThen, &BoolExpr{true})
		bf, df := v.visitExpr(s, bElse, &BoolExpr{false})
		// Join the branches
		be := newblock(b.Func, v.newlabel("end"))
		be.pred = append(be.pred, bt, bf)
		bt.succ = append(bt.succ, be)
		bf.succ = append(bf.succ, be)
		b.Func.blocks = append(b.Func.blocks, be)
		be.args = v.newreg1() // TODO: len(dt)?
		bt.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			Src:    dt,
		})
		bf.emit(Op{
			Opcode: JumpOp,
			Label:  []Label{be.name},
			Src:    df,
		})
		b, dst = be, be.args
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
		// Even if this is a comparison operator ("<")
		// we emit a BinOp, not a CompareOp.
		// the ir->asm pass interprets them differently:
		// BinOp "<" means to produce a boolean value;
		// CompareOp "<" means set the flags for a subsequent BranchOp
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
	case *TupleExpr:
		// evaluate the arguments
		var args = make([]Reg, len(e.Args))
		var tmp []Reg
		for i, a := range e.Args {
			b, tmp = v.visitExpr(s, b, a)
			args[i] = tmp[0]
		}
		// %n = len(args)
		n := v.newreg1()
		b.emit(Op{
			Opcode: LiteralOp,
			Dst:    n,
			Value:  int64(len(args)),
		})
		// %ptr = <pointer mask>
		ptr := v.newreg1()
		b.emit(Op{
			Opcode: LiteralOp,
			Dst:    ptr,
			Value:  int64(0), // TODO: need type information
		})
		// call newtuple
		dst = v.newreg1()
		b.emit(Op{
			Opcode:  CallOp, // primcall?
			Variant: "psc_newtuple",
			Dst:     dst,
			Src:     []Reg{n[0], ptr[0]},
		})
		// set tuple elements
		for i, a := range args {
			dst = v.newreg1()
			b.emit(Op{
				Opcode: RecordSetOp,
				Src:    []Reg{dst[0], a},
				Value:  uint64(i),
			})
		}
	case *TupleIndexExpr:
		var tu []Reg
		b, tu = v.visitExpr(s, b, e.Base)
		dst = v.newreg1()
		b.emit(Op{
			Opcode: RecordGetOp,
			Dst:    dst,
			Src:    tu,
			Value:  uint64(e.Index),
		})
	default:
		panic(fmt.Sprintf("unhandled case in visitExpr: %T", e))
	}
	return b, dst
}

func (e *BinExpr) isCompare() bool {
	switch e.Op {
	case "eq", "ne", "<", "<=", ">=", ">":
		return true
	default:
		return false
	}
}

func (v *compiler) visitCond(s *scope, b *block, e Expr) (blThen, blElse *block) {
	bThen := newblock(b.Func, v.newlabel("then"))
	bElse := newblock(b.Func, v.newlabel("else"))
	v.visitCond2(s, b, e, bThen, bElse)
	b.Func.blocks = append(b.Func.blocks, bThen, bElse)
	return bThen, bElse
}

func (v *compiler) visitCond2(s *scope, b *block, e Expr, bThen, bElse *block) {
	switch e := e.(type) {
	case *BoolExpr:
		// the textbook does something subtle here to avoid
		// adding the other block (bElse if true, bThen if false)
		// to the CFG at all, but i'm not that smart.
		// we can easily constant-fold if expressions in an
		// earlier pass or remove dead blocks in a later pass.
		// i think we're going to do a topological sort later
		// anyway, so it'll probably sort itself out.
		if e.Value == true {
			b.emit(Op{
				Opcode: JumpOp,
				Label:  []Label{bThen.name},
			})
			b.succ = append(b.succ, bThen)
			bThen.pred = append(bThen.pred, b)
		} else {
			b.emit(Op{
				Opcode: JumpOp,
				Label:  []Label{bElse.name},
			})
			b.succ = append(b.succ, bElse)
			bElse.pred = append(bElse.pred, b)
		}
	case *VarExpr:
		if s.has(e.Name) {
			ref := s.lookup(e.Name).(*mvar).Reg
			false := v.newreg()
			// Emit v == true
			// TODO: this should lower to orq a,a; jz
			b.emit(Op{
				Opcode: LiteralOp,
				Dst:    []Reg{false},
				Value:  int64(0),
			})
			cond := v.newreg1()
			b.emit(Op{
				Opcode:  CompareOp,
				Variant: "ne",
				Dst:     cond,
				Src:     []Reg{ref, false}, //XXX
			})
			// Emit branch
			b.emit(Op{
				Opcode: BranchOp,
				Src:    cond,
				Label:  []Label{bThen.name, bElse.name},
			})
			b.succ = append(b.succ, bThen, bElse)
			bThen.pred = append(bThen.pred, b)
			bElse.pred = append(bElse.pred, b)
		} else {
			v.errorf("%v is not in scope", e.Name)
		}
	case *BinExpr:
		if e.isCompare() {
			// Emit compare op
			b1, y := v.visitExpr(s, b, e.Left)
			b2, z := v.visitExpr(s, b1, e.Right)
			b = b2
			cond := v.newreg1()
			b.emit(Op{
				Opcode:  CompareOp,
				Variant: e.Op,
				Dst:     cond,
				Src:     []Reg{y[0], z[0]}, //XXX
			})
			// Emit branch
			b.emit(Op{
				Opcode: BranchOp,
				Src:    cond,
				Label:  []Label{bThen.name, bElse.name},
			})
			b.succ = append(b.succ, bThen, bElse)
			bThen.pred = append(bThen.pred, b)
			bElse.pred = append(bElse.pred, b)
		} else {
			v.errorf("cannot use non-boolean expression as condition: %v", e)
		}
	case *AndExpr:
		// if left is false, goto bElse
		// otherwise check right
		b2 := newblock(b.Func, v.newlabel("and"))
		v.visitCond2(s, b, e.Left, b2, bElse)
		v.visitCond2(s, b2, e.Right, bThen, bElse)
		b.Func.blocks = append(b.Func.blocks, b2)
	case *OrExpr:
		b2 := newblock(b.Func, v.newlabel("or"))
		v.visitCond2(s, b, e.Left, bThen, b2)
		v.visitCond2(s, b2, e.Right, bThen, bElse)
		b.Func.blocks = append(b.Func.blocks, b2)
	case *IfExpr:
		// this is an if embedded in the condition of another if.
		// its branches evaluate to booleans which become the condition
		// of the outer if.
		//
		innerThen, innerElse := v.visitCond(s, b, e.Cond)
		v.visitCond2(s, innerThen, e.Then, bThen, bElse)
		v.visitCond2(s, innerElse, e.Else, bThen, bElse)
	case *LetExpr:
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
		v.visitCond2(inner, b, e.Body, bThen, bElse)
	default:
		panic(fmt.Sprintf("unhandled case in visitCond: %T", e))
	}
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
