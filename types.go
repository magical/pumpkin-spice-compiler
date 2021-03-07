package main

import (
	"fmt"
	"strconv"
)

// types:
//   arbitrary-length signed integers
//   strings
//   boolean
//   functions
//   lists
//   bytes? - unsigned 8-bit value, with trapping overflow
//   no null
//   some sort of dict or object or table
//   any

type Type interface{}

type IntT struct{}
type StrT struct{}
type BoolT struct{}

type FuncT struct {
	Params []Type
	Return []Type
}

type ListT struct {
	Elem Type
}

type TupleT struct {
	Type []Type
}

type AnyT struct{}

func typecheck(e Expr) error {
	_, err := typecheck2(e)
	return err
}

func typecheck2(e Expr) (Type, error) {
	top := newscope(nil)
	top.vars["true"] = BoolT{}
	top.vars["false"] = BoolT{}
	return typecheckExpr(top, e)
}

func typecheckExpr(s *scope, expr Expr) (Type, error) {
	switch e := expr.(type) {
	case *VarExpr:
		if !s.has(e.Name) {
			return AnyT{}, fmt.Errorf("%v not in scope", e.Name)
		}
		return s.lookup(e.Name).(Type), nil
	case *IntExpr:
		return IntT{}, nil
	case *BoolExpr:
		return BoolT{}, nil
	case *BinExpr:
		t1, err1 := typecheckExpr(s, e.Left)
		t2, err2 := typecheckExpr(s, e.Right)
		if err1 != nil || err2 != nil {
			return AnyT{}, multiError(err1, err2)
		}
		var err error
		switch e.Op {
		case "+", "-", "*", "/":
			if !(t1 == IntT{} && t2 == IntT{}) {
				err = fmt.Errorf("operands to %s must be IntT, found %T and %T", e.Op, t1, t2)
			}
			return IntT{}, err
		case "<", "<=", ">=", ">":
			if !(t1 == IntT{} && t2 == IntT{}) {
				err = fmt.Errorf("operands to %s must be IntT, found %T and %T", e.Op, t1, t2)
			}
			return BoolT{}, err
		case "eq":
			if !comparableTypes(t1, t2) {
				err = fmt.Errorf("cannot compare %T and %T", t1, t2)
			}
			return BoolT{}, err
		default:
			panic(fmt.Sprintf("unhandled binop: %s", e.Op))
		}
	case *AndExpr:
		t1, err1 := typecheckExpr(s, e.Left)
		t2, err2 := typecheckExpr(s, e.Right)
		if err1 != nil || err2 != nil {
			return BoolT{}, multiError(err1, err2)
		}
		var err error
		if !(t1 == BoolT{} && t2 == BoolT{}) {
			err = fmt.Errorf("operands to 'and' must be BoolT, found %T and %T", t1, t2)
		}
		return BoolT{}, err
	case *OrExpr:
		t1, err1 := typecheckExpr(s, e.Left)
		t2, err2 := typecheckExpr(s, e.Right)
		if err1 != nil || err2 != nil {
			return BoolT{}, multiError(err1, err2)
		}
		var err error
		if !(t1 == BoolT{} && t2 == BoolT{}) {
			err = fmt.Errorf("operands to 'or' must be BoolT, found %T and %T", t1, t2)
		}
		return BoolT{}, err
	case *IfExpr:
		t1, err1 := typecheckExpr(s, e.Cond)
		t2, err2 := typecheckExpr(s, e.Then)
		t3, err3 := typecheckExpr(s, e.Else)
		if err1 == nil && (t1 != BoolT{}) {
			err1 = fmt.Errorf("if condition must be BoolT, found %T", t1)
		}
		if err2 == nil && err3 == nil {
			if sameType(t2, t3) {
				return t2, err1
			} else {
				err := fmt.Errorf("both branches of an if must have the same type, found %T and %T", t2, t3)
				return AnyT{}, multiError(err1, err)
			}
		} else {
			if sameType(t2, t3) {
				return t2, multiError(err1, err2, err3)
			} else {
				return AnyT{}, multiError(err1, err2, err3)
			}
		}
	case *LetExpr:
		inner := s.push()
		t1, err1 := typecheckExpr(s, e.Val)
		inner.vars[e.Var] = t1
		t2, err2 := typecheckExpr(inner, e.Body)
		return t2, multiError(err1, err2)
	case *FuncExpr:
		var params = make([]Type, len(e.Args))
		for i := range e.Args {
			params[i] = AnyT{} // XXX
		}
		inner := s.push()
		for i := range e.Args {
			inner.vars[e.Args[i]] = params[i]
		}
		if e.Name != "" && inner.vars[e.Name] == nil {
			inner.vars[e.Name] = FuncT{
				Params: params,
				// oh i guess we need to know the return type before typechecking the body,
				// at least for recursive functions
				Return: []Type{AnyT{}},
			}
		}
		rt, err := typecheckExpr(inner, e.Body) // TODO: multiple returns?
		return FuncT{Params: params, Return: []Type{rt}}, err
	case *CallExpr:
		var errors []error
		if v, ok := e.Func.(*VarExpr); ok {
			if !s.has(v.Name) && isBuiltin(v.Name) {
				return typecheckBuiltin(v.Name, e.Args, s)
			}
		}
		// get the function type
		t1, err1 := typecheckExpr(s, e.Func)
		if _, ok := t1.(FuncT); !ok {
			if err1 != nil {
				return AnyT{}, err1
			} else {
				// TODO: allow calling AnyT
				return AnyT{}, fmt.Errorf("cannot call non-function type %T", t1)
			}
		}
		// get the argument types
		f := t1.(FuncT)
		args := make([]Type, len(e.Args))
		for i := range e.Args {
			var err error
			args[i], err = typecheckExpr(s, e.Args[i])
			errors = append(errors, err)
		}
		// check arguments against parameter types
		if len(f.Params) != len(e.Args) {
			errors = append(errors, fmt.Errorf("function has %d arguments, found %d", len(f.Params), len(e.Args)))
		}
		for i := 0; i < len(f.Params) && i < len(args); i++ {
			// TODO: don't add this if the argument fail to typecheck
			if !sameType(f.Params[i], args[i]) {
				errors = append(errors, fmt.Errorf("argument %d is %T, found %T", i, f.Params[i], args[i]))
			}
		}
		if len(f.Return) > 1 {
			errors = append(errors, fmt.Errorf("function with mulitple return values used in a single-value context"))
		}
		if len(f.Return) == 0 {
			errors = append(errors, fmt.Errorf("function with no return value used as an expression"))
			return AnyT{}, multiError(errors...)
		}
		return f.Return[0], multiError(errors...)
	case *DotExpr:
		return AnyT{}, fmt.Errorf("TODO dot expression")
	default:
		panic(fmt.Sprintf("unhandled case: %T", e))
	}
}

func isBuiltin(s string) bool {
	switch s {
	case "tuple", "get":
		return true
	default:
		return false
	}
}

func typecheckBuiltin(name string, args []Expr, s *scope) (Type, error) {
	switch name {
	case "tuple":
		var types = make([]Type, len(args))
		var errors []error
		for i := range args {
			var err error
			types[i], err = typecheckExpr(s, args[i])
			if err != nil {
				errors = append(errors, err)
			}
		}
		return TupleT{Type: types}, multiError(errors...)
	case "get":
		if len(args) != 2 {
			// TODO: still typecheck first arg, if present?
			return AnyT{}, fmt.Errorf("get takes 2 arguments, found %d", len(args))
		}
		t, err := typecheckExpr(s, args[0])
		if err != nil {
			return AnyT{}, err
		}
		if !isTupleT(t) {
			return AnyT{}, fmt.Errorf("first argument to 'get' must be a tuple, found %T", t)
		}
		// *don't* typecheck the second arg; it must be a literal
		if !isInt(args[1]) {
			return AnyT{}, fmt.Errorf("second argument to 'get' must be an integer literal, found %#v", args[1])
		}
		n, err := strconv.Atoi(args[1].(*IntExpr).Value)
		if err != nil {
			fatalf("couldn't parse tuple index: %v", err)
		}
		return t.(TupleT).Type[n], nil
	default:
		fatalf("unknown builtin %s", name)
	}
	panic("unreachable")
}

// aggregates multiple errors.
// strips out nils (may modify the input list).
func multiError(errors ...error) error {
	j := 0
	for i := range errors {
		if errors[i] != nil {
			if i != j {
				errors[j] = errors[i]
			}
			j++
		}
	}
	switch j {
	case 0:
		return nil
	case 1:
		return errors[0]
	default:
		return ErrorList(errors[:j])
	}
}

func comparableTypes(t1, t2 Type) bool {
	switch {
	case t1 == IntT{} && t2 == IntT{}:
		return true
	case t1 == BoolT{} && t2 == BoolT{}:
		return true
	case t1 == AnyT{}:
		switch t2.(type) {
		case AnyT, IntT, BoolT:
			return true
		default:
			return false
		}
	case t2 == AnyT{}:
		switch t1.(type) {
		case AnyT, IntT, BoolT:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func sameType(t1, t2 Type) bool {
	return t1 == t2
}

func isTupleT(t Type) bool {
	_, ok := t.(TupleT)
	return ok
}

// Typecheck decorates a Prog with types.
// It returns any type errors encountered.
func Typecheck(*Prog) error {
	return nil
}
