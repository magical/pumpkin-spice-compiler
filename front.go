// front-end passes
//
// * uncover booleans
// * uncover tuples

package main

import (
	"fmt"
	"strconv"
)

// the uncover-bools pass replaces any unshadowed "true" or "false" variables
// with a BoolExpr
func uncoverBools(e Expr) Expr {
	var top scope
	return uncoverBoolsExpr(&top, e)
}

func uncoverBoolsExpr(s *scope, expr Expr) Expr {
	// the VarExpr case is the only one that does any work
	// the rest just propagate the recursion
	switch e := expr.(type) {
	case *VarExpr:
		if !s.has(e.Name) {
			if e.Name == "true" {
				return &BoolExpr{true}
			}
			if e.Name == "false" {
				return &BoolExpr{false}
			}
		}
		break
	case *BoolExpr:
		break
	case *IntExpr:
		break
	case *DotExpr:
		return &DotExpr{
			Left:  uncoverBoolsExpr(s, e.Left),
			Right: e.Right,
		}
	case *BinExpr:
		return &BinExpr{
			Op:    e.Op,
			Left:  uncoverBoolsExpr(s, e.Left),
			Right: uncoverBoolsExpr(s, e.Right),
		}
	case *AndExpr:
		return &AndExpr{
			Left:  uncoverBoolsExpr(s, e.Left),
			Right: uncoverBoolsExpr(s, e.Right),
		}
	case *OrExpr:
		return &OrExpr{
			Left:  uncoverBoolsExpr(s, e.Left),
			Right: uncoverBoolsExpr(s, e.Right),
		}
	case *CallExpr:
		var args = make([]Expr, len(e.Args))
		for i := range e.Args {
			args[i] = uncoverBoolsExpr(s, e.Args[i])
		}
		return &CallExpr{
			Func: uncoverBoolsExpr(s, e.Func),
			Args: args,
		}
	case *LetExpr:
		inner := s.push()
		inner.define(e.Var)
		return &LetExpr{
			Var:  e.Var,
			Val:  uncoverBoolsExpr(s, e.Val),
			Body: uncoverBoolsExpr(inner, e.Body),
		}
	case *IfExpr:
		return &IfExpr{
			Cond: e.Cond,
			Then: uncoverBoolsExpr(s, e.Then),
			Else: uncoverBoolsExpr(s, e.Else),
		}
	case *FuncExpr:
		inner := s.push()
		for _, p := range e.Args {
			inner.define(p)
		}
		return &FuncExpr{
			Name: e.Name,
			Args: e.Args,
			Body: uncoverBoolsExpr(s, e.Body),
		}
	default:
		panic(fmt.Sprintf("unhandled case: %T", e))
	}
	return expr
}

// the uncover-tuples pass replaces tuple(..) with TupleExpr
// and get(x, n) with TupleIndexExpr
// TODO: prim.tuple and prim.get?
func uncoverTuples(e Expr) Expr {
	var top scope
	return uncoverTuplesExpr(&top, e)
}

func uncoverTuplesExpr(s *scope, expr Expr) Expr {
	// the VarExpr case is the only one that does any work
	// the rest just propagate the recursion
	switch e := expr.(type) {
	case *VarExpr:
		if !s.has(e.Name) {
			if e.Name == "tuple" {
				fmt.Println("error: tuple in non-call context")
			}
			if e.Name == "get" {
				fmt.Println("error: get in non-call context")
			}
		}
		break
	case *BoolExpr:
		break
	case *IntExpr:
		break
	case *DotExpr:
		return &DotExpr{
			Left:  uncoverTuplesExpr(s, e.Left),
			Right: e.Right,
		}
	case *BinExpr:
		return &BinExpr{
			Op:    e.Op,
			Left:  uncoverTuplesExpr(s, e.Left),
			Right: uncoverTuplesExpr(s, e.Right),
		}
	case *AndExpr:
		return &AndExpr{
			Left:  uncoverTuplesExpr(s, e.Left),
			Right: uncoverTuplesExpr(s, e.Right),
		}
	case *OrExpr:
		return &OrExpr{
			Left:  uncoverTuplesExpr(s, e.Left),
			Right: uncoverTuplesExpr(s, e.Right),
		}
	case *CallExpr:
		if e := uncoverTupleBuiltins(s, e); e != nil {
			return e
		}
		var args = make([]Expr, len(e.Args))
		for i := range e.Args {
			args[i] = uncoverTuplesExpr(s, e.Args[i])
		}
		return &CallExpr{
			Func: uncoverTuplesExpr(s, e.Func),
			Args: args,
		}
	case *LetExpr:
		inner := s.push()
		inner.define(e.Var)
		return &LetExpr{
			Var:  e.Var,
			Val:  uncoverTuplesExpr(s, e.Val),
			Body: uncoverTuplesExpr(inner, e.Body),
		}
	case *IfExpr:
		return &IfExpr{
			Cond: e.Cond,
			Then: uncoverTuplesExpr(s, e.Then),
			Else: uncoverTuplesExpr(s, e.Else),
		}
	case *FuncExpr:
		inner := s.push()
		for _, p := range e.Args {
			inner.define(p)
		}
		return &FuncExpr{
			Name: e.Name,
			Args: e.Args,
			Body: uncoverTuplesExpr(s, e.Body),
		}
	default:
		panic(fmt.Sprintf("unhandled case: %T", e))
	}
	return expr
}

func uncoverTupleBuiltins(s *scope, e *CallExpr) Expr {
	v, ok := e.Func.(*VarExpr)
	if !ok || s.has(v.Name) {
		return nil
	}
	switch v.Name {
	case "tuple":
		return &TupleExpr{
			Args: e.Args,
		}
	case "get":
		if len(e.Args) == 2 && isInt(e.Args[1]) {
			n, _ := strconv.Atoi(e.Args[1].(*IntExpr).Value)
			return &TupleIndexExpr{
				Base:  e.Args[0],
				Index: n,
			}
		}
	}
	return e
}

func isInt(e Expr) bool {
	_, ok := e.(*IntExpr)
	return ok
}
