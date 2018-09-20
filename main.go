package main

import (
	"io"
	"os"
)

func main() {
	parse(os.Stdin)
}

func parse(r io.Reader) {
	l := new(lexer)
	l.Init(r)
	yyParse(l)
}
