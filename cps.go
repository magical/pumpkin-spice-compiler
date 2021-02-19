package main

import "fmt"

// cps.go does an ast-to-ast transformation
// from arbitrary source code into continuation passing style

// C[k] [ func(...) body end ] = func(k1, ...) k1(C[k][ body ]) end
// C[k] [ let v = x in body end ] = C[ func(v) C[k][ body ] ][ x ]
// C[k] [ expr ] = k(expr)

func cpsConvert(k, expr Expr) Expr {
	switch e := expr.(type) {
	case *VarExpr:
		return &CallExpr{Func: k, Args: []Expr{e}}
	case *IntExpr:
		return &CallExpr{Func: k, Args: []Expr{e}}
	case *DotExpr:
		return &CallExpr{Func: k, Args: []Expr{e}}
	case *BinExpr:
		// assume the operands are trivial
		return &CallExpr{Func: k, Args: []Expr{e}}
	case *CallExpr:
		// arguments cannot be function calls
		// maybe this shoud be a separate pass?
		// yeah, just assume that's already been done
		//
		// add continuation parameter
		return &CallExpr{
			Func: e.Func,
			//Args: append([]Expr{k}, e.Args...),
			Args: append(e.Args[:len(e.Args):len(e.Args)], k),
		}
	case *LetExpr:
		// construct a continuation
		k1 := &FuncExpr{
			Args: []string{e.Var},
			Body: cpsConvert(k, e.Body),
		}
		return cpsConvert(k1, e.Val)
	case *IfExpr:
		return &IfExpr{
			Cond: e.Cond,
			Then: cpsConvert(k, e.Then),
			Else: cpsConvert(k, e.Else),
		}
	case *FuncExpr:
		// add continuation argument
		f := &FuncExpr{Name: e.Name,
			//Args: append([]string{"k"}, e.Args...),
			Args: append(e.Args[:len(e.Args):len(e.Args)], "k"),
			Body: cpsConvert(&VarExpr{"k"}, e.Body),
		}
		return &CallExpr{Func: k, Args: []Expr{f}}
	default:
		panic(fmt.Sprintf("unhandled case: %T", e))
	}
}

func isCall(e Expr) bool {
	_, ok := e.(*CallExpr)
	return ok
}

// a trivial expr is one that can be computed without making any function calls
func isTrivial(e Expr) bool {
	switch e := e.(type) {
	case *VarExpr:
		return true
	case *IntExpr:
		return true
	case *BinExpr:
		return isTrivial(e.Left) && isTrivial(e.Right)
	case *CallExpr:
		return false
	case *DotExpr:
		return isTrivial(e.Left)
	case *LetExpr:
		return isTrivial(e.Val) && isTrivial(e.Body)
	case *IfExpr:
		return isTrivial(e.Then) && isTrivial(e.Else)
	case *FuncExpr:
		return true // !!
	default:
		panic(fmt.Sprintf("unhandled case in isTrivial: %T", e))
	}
}

// Generic visit function, for eas of copy-pasting...
func visitSkeleton(expr Expr) Expr {
	switch e := expr.(type) {
	case *VarExpr:
	case *IntExpr:
	case *BinExpr:
	case *CallExpr:
	case *DotExpr:
	case *LetExpr:
	case *IfExpr:
	case *FuncExpr:
	default:
		panic(fmt.Sprintf("unhandled case: %T", e))
	}
	return nil
}
