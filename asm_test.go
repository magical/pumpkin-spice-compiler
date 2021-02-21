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
			mkinstr("movq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("addq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("subq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("cmpq", mkmem("rbx", 0), mkmem("rbx", 4)),
			mkinstr("retq"),
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
			mkinstr("retq"),
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

func printAsmBlock(b *asmBlock) {
	var p AsmPrinter
	p.w = os.Stdout
	p.ConvertBlock(b)

}
