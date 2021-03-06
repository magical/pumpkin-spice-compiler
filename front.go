// front-end passes
//
// * uncover booleans
// * uncover tuples

package main

import "fmt"

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
