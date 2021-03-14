package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestAsmPrinter(t *testing.T) {
	block := asmBlock{
		label: "0",
		code: []asmOp{
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, {Imm: 10}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rax"}, {Imm: 2}}},
			{tag: asmInstr, variant: "subq", args: []asmArg{{Reg: "rax"}, {Imm: 2}}},
			{tag: asmInstr, variant: "negq", args: []asmArg{{Reg: "rax"}}},
		},
	}
	want := `
	.globl psc_main
psc_main:
	pushq %rbp
	movq %rsp, %rbp
	pushq  %r15
	movq   $4096,%rsi
	movq   $4096,%rdi
	callq  psc_gcinit
	movq   rootstack_begin(%rip),%r15
.L0:
	movq $10, %rax
	addq $2, %rax
	subq $2, %rax
	negq %rax

	popq %r15
	popq %rbp
	ret
`

	var p AsmPrinter
	buf := new(bytes.Buffer)
	p.w = buf
	p.convertSingleBlockProgram(&block)
	got := buf.String()
	if got != want {
		t.Errorf("asm output didn't match\nexpected:\n%s\nactual:\n%s", want, got)
	}
}

func TestPatchInstructions(t *testing.T) {
	var rax = asmArg{Reg: "rax"}
	block := asmBlock{
		code: []asmOp{
			// these are rewritten
			mkinstr("movq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("addq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("subq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("cmpq", mkmem("rbx", 0), mkmem("rbx", 4)),
			// these are not
			mkinstr("movq", rax, rax),
			mkinstr("movq", rax, asmArg{Imm: 42}),
			mkinstr("ret"),
		},
	}
	want := asmBlock{
		code: []asmOp{
			mkinstr("movq", rax, mkmem("rbx", 4)),
			mkinstr("movq", mkmem("rbx", 0), rax),
			mkinstr("movq", rax, mkmem("rbx", 4)),
			mkinstr("addq", mkmem("rbx", 0), rax),
			mkinstr("movq", rax, mkmem("rbx", 4)),
			mkinstr("subq", mkmem("rbx", 0), rax),
			mkinstr("movq", rax, mkmem("rbx", 4)),
			mkinstr("cmpq", mkmem("rbx", 0), rax),
			//---
			mkinstr("movq", rax, rax),
			mkinstr("movq", rax, asmArg{Imm: 42}),
			mkinstr("ret"),
		},
	}

	block.patchInstructions()
	if !reflect.DeepEqual(&block, &want) {
		fmt.Println("got:")
		printAsmBlock(&block)
		fmt.Println("want:")
		printAsmBlock(&want)
		t.Errorf("got %+v, want %+v", &block, &want)
	}
}

func TestAssignHomes(t *testing.T) {
	block := &asmBlock{
		label: "L0",
		code: []asmOp{
			{tag: asmInstr, variant: "movq", args: []asmArg{{Var: "x"}, {Imm: 20}}},
			{tag: asmInstr, variant: "movq", args: []asmArg{{Var: "y"}, {Imm: 2}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Var: "x"}, {Var: "x"}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Var: "x"}, {Var: "y"}}},
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, {Var: "x"}}},
		},
	}
	if err := block.checkMachineInstructions(); err != nil {
		t.Error(err)
	}
	prog := &asmProg{
		blocks: []*asmBlock{block},
	}
	prog.assignHomes(nil)
	prog.addStackFrameInstructions()

	var expected *asmBlock
	var wantstacksize int
	if useFancyAllocator {
		expected = &asmBlock{
			label: "L0",
			args:  nil,
			code: []asmOp{
				{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rcx"}, {Imm: 20}}},
				{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rdx"}, {Imm: 2}}},
				{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rcx"}, {Reg: "rcx"}}},
				{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rcx"}, {Reg: "rdx"}}},
				{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, {Reg: "rcx"}}},
			},
		}
		wantstacksize = 0
	} else {
		expected = &asmBlock{
			label: "L0",
			args:  nil,
			code: []asmOp{
				{tag: asmInstr, variant: "subq", args: []asmArg{{Reg: "rsp"}, {Imm: 16}}},
				{tag: asmInstr, variant: "movq", args: []asmArg{mkmem("rsp", 0), {Imm: 20}}},
				{tag: asmInstr, variant: "movq", args: []asmArg{mkmem("rsp", 8), {Imm: 2}}},
				{tag: asmInstr, variant: "addq", args: []asmArg{mkmem("rsp", 0), mkmem("rsp", 0)}},
				{tag: asmInstr, variant: "addq", args: []asmArg{mkmem("rsp", 0), mkmem("rsp", 8)}},
				{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, mkmem("rsp", 0)}},
				{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rsp"}, {Imm: 16}}},
			},
		}
		wantstacksize = 16
	}
	if !reflect.DeepEqual(block, expected) {
		fmt.Println("got:")
		printAsmBlock(block)
		fmt.Println("want:")
		printAsmBlock(expected)
		t.Errorf("got %+v, want %+v", block, expected)
	}
	if prog.stacksize != wantstacksize {
		t.Errorf("got stacksize = %d, want %d", prog.stacksize, wantstacksize)
	}
}

func printAsmBlock(b *asmBlock) {
	var p AsmPrinter
	p.w = os.Stdout
	p.ConvertBlock(b)
}

func TestCompile(t *testing.T) {
	if !useFancyAllocator {
		t.Skip("fancy allocator not enabled")
	}
	const source = `let v = 1 in let w = 42 in let x = v + 7 in let y = x in let z = x + w in z - y end end end end end`
	const want = asmPrologue +
		`.Lentry:
	movq $1, %rcx
	addq $7, %rcx
	movq %rcx, %rdx
	addq $42, %rdx
	subq %rcx, %rdx
	movq %rdx, %rax
` + asmEpilogue

	expr, err := parse(strings.NewReader(source))
	if err != nil {
		t.Fatal("parse failed: ", err)
	}
	prog := lower(expr)
	block := prog.funcs[0].blocks[0]
	b := block.SelectInstructions(prog.funcs[0])
	if err := b.checkMachineInstructions(); err != nil {
		printAsmBlock(b)
		t.Fatal("chechMachineInstructions failed: ", err)
	}

	p := &asmProg{blocks: []*asmBlock{b}}
	p.assignHomes(nil)
	p.addStackFrameInstructions()

	b.patchInstructions()

	var pr AsmPrinter
	buf := new(bytes.Buffer)
	pr.w = buf
	pr.convertSingleBlockProgram(b)

	got := buf.String()

	if want != got {
		t.Errorf("want:%s\ngot:%s", want, got)
	}
}
