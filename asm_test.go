package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"
)

var asmTests = []struct {
	block  asmBlock
	output string
}{
	{asmBlock{
		label: "L0",
		code: []asmOp{
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, {Imm: 10}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rax"}, {Imm: 2}}},
		},
	}, `
	.globl psc_main
psc_main:
	pushq %rbp
	movq %rsp, %rbp
.L0:
	movq $10, %rax
	addq $2, %rax

	popq %rbp
	ret
`}}

func TestAsmPrinter(t *testing.T) {
	for _, tt := range asmTests {
		var p AsmPrinter
		buf := new(bytes.Buffer)
		p.w = buf
		p.ConvertBlock(&tt.block)
		got := buf.String()
		if got != tt.output {
			t.Errorf("output didn't match\nblock: %+v\nexpected:\n%s\nactual:\n%s", tt.block, tt.output, got)
		}
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
	block.assignHomes()
	block.addStackFrameInstructions()

	expected := &asmBlock{
		label: "L0",
		args:  nil,
		code: []asmOp{
			//{tag: asmInstr, variant: "subq", args: []asmArg{{Reg: "rsp"}, {Imm: 16}}},
			//{tag: asmInstr, variant: "movq", args: []asmArg{mkmem("rsp", 0), {Imm: 20}}},
			//{tag: asmInstr, variant: "movq", args: []asmArg{mkmem("rsp", 8), {Imm: 2}}},
			//{tag: asmInstr, variant: "addq", args: []asmArg{mkmem("rsp", 0), mkmem("rsp", 0)}},
			//{tag: asmInstr, variant: "addq", args: []asmArg{mkmem("rsp", 0), mkmem("rsp", 8)}},
			//{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, mkmem("rsp", 0)}},
			//{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rsp"}, {Imm: 16}}},
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rcx"}, {Imm: 20}}},
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rdx"}, {Imm: 2}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rcx"}, {Reg: "rcx"}}},
			{tag: asmInstr, variant: "addq", args: []asmArg{{Reg: "rcx"}, {Reg: "rdx"}}},
			{tag: asmInstr, variant: "movq", args: []asmArg{{Reg: "rax"}, {Reg: "rcx"}}},
		},
		stacksize: 0,
	}
	if !reflect.DeepEqual(block, expected) {
		fmt.Println("got:")
		printAsmBlock(block)
		fmt.Println("want:")
		printAsmBlock(expected)
		t.Errorf("got %+v, want %+v", block, expected)
	}
}

func printAsmBlock(b *asmBlock) {
	var p AsmPrinter
	p.w = os.Stdout
	p.ConvertBlock(b)
}
