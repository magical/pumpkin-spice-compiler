package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kr/pretty"
)

func main() {
	if err := main2(); err != nil {
		fmt.Println(err)
	}
}

func main2() error {
	x, err := parse(os.Stdin)
	if err != nil {
		return nil
	}
	printExpr(x)
	fmt.Println("=======")
	y := cpsConvert(&VarExpr{"return"}, x)
	printExpr(y)
	return err
}

func main1() error {
	// fun test expressions:
	//
	// a+b*c(d(),)
	// (func(a) let f = func(b) 5 + a end in f(a) end end)(a)
	//

	x, err := parse(os.Stdin)
	if err != nil {
		return err
	}
	pretty.Println(x)
	printExpr(x)
	y := lower(x)
	pretty.Println(y)
	print(y)

	/*
		prog := &Prog{
			blocks: []*block{
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
	*/
	return nil
}

type ErrorList []error

func (list ErrorList) Error() string {
	var b []byte
	for i, err := range list {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, []byte(err.Error())...)
	}
	return string(b)
}

func parse(r io.Reader) (Expr, error) {
	l := new(lexer)
	l.Init(r)
	yyParse(l)
	var err error
	if len(l.errors) > 0 {
		err = ErrorList(l.errors)
	}
	return l.result, err
}
