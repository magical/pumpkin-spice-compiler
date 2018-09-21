package main

// lower.go is the middle-end of the compiler
// it takes a top-level Expr and lowers it down to a Prog

type Prog struct {
	funcs []*Proc
}

type Proc struct {
	name string
	code []Lop
}

type Reg string

type Lopcode int

const (
	Lnoop Lopcode = iota

	Linit // a = k
	Lcopy // a = b
	Ladd  // a = b + c
	Lsub  // a = b - c
	Lmul  // a = b * c
	Ldiv  // a = b / c
	Lcall // b(c)
)

// A lop is a low-level operation
type Lop struct {
	Op      Lopcode
	A, B, C Reg
	K       int
}
