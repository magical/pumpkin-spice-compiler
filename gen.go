package main

import (
	"bytes"
	"fmt"
	"strings"
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
	for _, b := range p.blocks {
		genb(&w, b)
	}
	return w.String()
}

func genb(w *bytes.Buffer, b *block) {
	// signature
	w.WriteString("void ")
	w.WriteString(string(b.name))
	w.WriteString("() {\n")
	// variables
	// TODO: sort
	for name, _ := range b.scope.vars {
		fmt.Fprintf(w, "    int %s;\n", string(name))
	}
	// code
	for _, l := range b.code {
		genl(w, l)
	}
	w.WriteString("}\n")
}

func genl(w *bytes.Buffer, l Op) {
	switch l.Opcode {
	case Noop:
	case LiteralOp:
		fmt.Fprintf(w, "    %s = %v;\n", l.Dst[0], l.Value)
	//case Lcopy:
	//	fmt.Fprintf(w, "    %s = %s;\n", l.Dst[0], l.Src[0])
	case BinOp:
		fmt.Fprintf(w, "    %s = %s %s %s;\n", l.Dst[0], l.Src[0], l.Variant, l.Src[1])
	case CallOp:
		dst := l.Dst[0] // XXX
		var regs = make([]string, len(l.Src)-1)
		for i, r := range l.Src[1:] {
			regs[i] = string(r)
		}
		args := strings.Join(regs, ",")
		fmt.Fprintf(w, "    %s = %s(%s);\n", dst, l.Src[0], args)
	default:
		fmt.Println("unknown opcode:", l.Opcode)
		panic("unreachable")
	}
}
