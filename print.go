package main

import (
	"bytes"
	"fmt"
)

// this file pretty-prints Progs and blocks for debugging

func print(p *Prog) {
	for _, f := range p.funcs {
		fmt.Printf("FUNCTION %s\n", f.Name)
		for _, b := range f.blocks {
			fmt.Printf("  %s", b.name)
			if len(b.args) > 0 {
				fmt.Printf("(")
				for i, r := range b.args {
					if i != 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%%%s", r)
				}
				fmt.Printf(")")
			}
			fmt.Printf(":\n")
			printb(b)
		}
	}
}

func printb(b *block) {
	var buf bytes.Buffer
	for i, l := range b.code {
		fmt.Printf("\t%3d: %s\n", i, l.debugstr(&buf))
	}
}

func (l Op) debugstr(b *bytes.Buffer) string {
	b.Reset()
	if len(l.Dst) > 0 {
		for i, r := range l.Dst {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString("%" + string(r))
		}
		b.WriteString(" = ")
	}

	b.WriteString(l.Opcode.String())

	if l.Variant != "" {
		b.WriteString(" \"")
		b.WriteString(l.Variant)
		b.WriteString("\"")
	}

	if len(l.Src) > 0 {
		b.WriteString(" ")
		for i, r := range l.Src {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString("%" + string(r))
		}
	}

	if len(l.Label) > 0 {
		b.WriteString(" {")
		for i, d := range l.Label {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(string(d))
		}
		b.WriteString("}")
	}

	if l.Value != nil {
		fmt.Fprint(b, " <", l.Value, ">")
	}
	return b.String()
}
