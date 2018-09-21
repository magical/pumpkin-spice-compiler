package main

import (
	"bytes"
	"fmt"
)

// bytecode:
//  a = b
//  a = add b c
//  a = sub b c
//  a = mul b c
//  a = div b c
//  a = call f x

func gen(p *Prog) string {
	var w bytes.Buffer
	for _, f := range p.funcs {
		genf(&w, f)
	}
	return w.String()
}

func genf(w *bytes.Buffer, f *Proc) {
	// signature
	w.WriteString("void ")
	w.WriteString(f.name)
	w.WriteString("() {\n")
	// variables
	seen := make(map[string]bool)
	declare := func(name Reg) {
		if name != "" && !seen[string(name)] {
			fmt.Fprintf(w, "    int %s;\n", string(name))
			seen[string(name)] = true
		}
	}
	for _, l := range f.code {
		declare(l.A)
		declare(l.B)
		declare(l.C)
	}

	// code
	for _, l := range f.code {
		genl(w, l)
	}
	w.WriteString("}\n")
}

var lbinop = map[Lopcode]string{
	Ladd: "+",
	Lsub: "-",
	Lmul: "*",
	Ldiv: "/",
}

func genl(w *bytes.Buffer, l Lop) {
	switch l.Op {
	case Lnoop:
	case Linit:
		fmt.Fprintf(w, "    %s = %d;\n", l.A, l.K)
	case Lcopy:
		fmt.Fprintf(w, "    %s = %s;\n", l.A, l.B)
	case Ladd, Lsub, Lmul, Ldiv:
		fmt.Fprintf(w, "    %s = %s %s %s;\n", l.A, l.B, lbinop[l.Op], l.C)
	case Lcall:
		fmt.Fprintf(w, "    %s(%s);\n", l.B, l.C)
	default:
		fmt.Println("unknown opcode:", l.Op)
		panic("unreachable")
	}
}
