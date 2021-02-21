package main

import (
	"fmt"
	"io"
	"strconv"
)

// asm converts ops into assembly code

// reg::=rsp|rbp|rax|rbx|rcx|rdx|rsi|rdi|r8|r9|r10|r11|r12|r13|r14|r15
// arg::=$int|%reg|int(%reg)
// instr::=addq arg,arg | subq arg,arg | negq arg | movq arg,arg
//       | callq label | pushq arg | popq arg | retq
//       | jmp label | label: instr

type AsmPrinter struct {
	w io.Writer
}

const asmPrologue = `
	.globl psc_main
psc_main:
`

const asmEpilogue = `
	ret
`

func (pr *AsmPrinter) ConvertProg(p *Prog) {
	panic("TODO")
}

func (pr *AsmPrinter) ConvertBlock(b *asmBlock) {
	io.WriteString(pr.w, asmPrologue)
	pr.write("." + string(b.label) + ":\n")
	if len(b.args) > 0 {
		fatalf("block with nonzero args: %+v", b)
	}
	for _, l := range b.code {
		switch l.tag {
		case asmInstr:
			pr.write("\t" + l.asmInstr() + "\n")
		case asmJump:
			pr.write("\tjmp ." + string(l.label) + "\n")
		case asmCall:
			pr.write("\tcallq " + string(l.label) + "\n")
		}
	}
	io.WriteString(pr.w, asmEpilogue)
}

func (pr *AsmPrinter) write(s string) {
	io.WriteString(pr.w, s)
}

func (l *asmOp) asmInstr() string {
	if len(l.args) == 0 {
		return l.variant
	}
	s := l.variant + " "
	for _, a := range l.args[1:] {
		s += a.String() + ", "
	}
	// first argument is the destination argument
	// but it goes last in att-style assembly
	s += l.args[0].String()
	return s
}

func fatalf(s string, args ...interface{}) {
	msg := fmt.Sprintf(s, args...)
	panic("fatal compile error: " + msg)
}

// An asmblock is a non-portable representation of a group of assembly instructions.
type asmBlock struct {
	label asmLabel
	args  []asmArg
	code  []asmOp
}

// An asmOp represents an x86-64 assembly instruction
type asmOp struct {
	tag     asmTag
	variant string
	args    []asmArg
	label   asmLabel
	// type information?
	// line number?
}

const (
	_ asmTag = iota

	asmInstr // $variant arg, arg
	asmCall  // call label (int?)
	//asmRet   // ret
	//asmPush  // push arg
	//asmPop   // pop arg
	asmJump // jmp labe
)

type asmTag int
type asmLabel string

// register, immediate, or offset from a register
type asmArg struct {
	Reg   string // rax, rbx, ... r10, r11 etc
	Imm   int64
	Deref bool
}

func (a asmArg) String() string {
	if a.Deref {
		return fmt.Sprintf("%d(%%%s)", a.Imm, a.Reg)
	} else if a.Reg != "" {
		return "%" + a.Reg
	} else {
		return "$" + strconv.FormatInt(a.Imm, 10)
	}
}

type AsmPasses struct{}

// The patch instructions pass fixes up instructions like
// 	movq 0(%r1), 1(%r2)
// which have too many memory references by adding an intermediate
// instruction which loads one of the memory locations into a register.
// 	movq %rax, 1(%r2)
// 	movq 0(%r1), %rax
// We assume %rax is available as a scratch register
func (_ *AsmPasses) patchInstructions(b *asmBlock) {
	b.patchInstructions()
}

func (b *asmBlock) patchInstructions() {
	var rax = asmArg{Reg: "rax"}
	for i := 0; i < len(b.code); i++ {
		l := b.code[i]
		switch l.tag {
		case asmInstr:
			// we don't have any instrucions that take more than
			// one argument yet, so we can just check for 2
			if len(l.args) == 2 && l.args[0].isMem() && l.args[1].isMem() {
				// make space for another op
				b.code = append(b.code[:i+1], b.code[i:]...)
				b.code[i] = mkinstr("movq", rax, l.args[1])
				b.code[i+1] = mkinstr(l.variant, l.args[0], rax)
				i++
			}
		}
	}
}

func mkinstr(variant string, args ...asmArg) asmOp {
	return asmOp{
		tag:     asmInstr,
		variant: variant,
		args:    args,
	}
}

func mkmem(reg string, offset int64) asmArg {
	return asmArg{Reg: reg, Imm: offset, Deref: true}
}

func (a *asmArg) isMem() bool { return a.Deref }
