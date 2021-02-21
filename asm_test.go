package main

import (
	"bytes"
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
.L0:
	movq 10, %rax
	addq 2, %rax

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
