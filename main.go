package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	parse(os.Stdin)

	prog := &Prog{
		funcs: []*Proc{
			{
				code: []Lop{
					{Op: Linit, A: "b", K: 6},
					{Op: Linit, A: "c", K: 2},
					{Op: Ladd, A: "a", B: "b", C: "c"},
				},
			},
		},
	}
	fmt.Println(gen(prog))
}

func parse(r io.Reader) {
	l := new(lexer)
	l.Init(r)
	yyParse(l)
}
