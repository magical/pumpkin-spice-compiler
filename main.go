package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kr/pretty"
)

func main() {
	// fun test expressions:
	//
	// a+b*c(d(),)
	// (func(a) let f = func(b) 5 + a end in f(a) end end)(a)
	//

	x := parse(os.Stdin)
	pretty.Println(x)
	y := lower(x)
	pretty.Println(y)

	prog := &Prog{
		blocks: []*Block{
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
