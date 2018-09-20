package main

import "bytes"

func gen(expr Expr) string {
	var w bytes.Buffer
	genw(&w, expr)
	return w.String()
}

func genw(w *bytes.Buffer, expr Expr) {
	switch v := expr.(type) {
	case *Func:
		w.WriteString("void ")
		w.WriteString(v.Name)
		w.WriteString("() {\n return ")
		genw(w, v.Body)
		w.WriteString(";\n}\n")
	case *BinExpr:
		genw(w, v.Left)
		w.WriteString(v.Op)
		genw(w, v.Right)
	}
}
