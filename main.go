package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kr/pretty"
)

func main() {
	x := parse(os.Stdin)
	pretty.Println(x)

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

func parse(r io.Reader) Expr {
	l := new(lexer)
	l.Init(r)
	yyParse(l)
	return l.result
}
