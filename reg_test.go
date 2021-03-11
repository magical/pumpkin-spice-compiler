package main

import (
	"fmt"
	"testing"
)

func TestRegalloc_Div(t *testing.T) {
	// let a = 1+0 in let b = 2+0 in let c = 3+0 in -b + (4*a*c - b*b)/(2*a) end end end
	var entry = block{
		name: "entry",
		code: []Op{
			{Opcode: LiteralOp, Dst: []Reg{"r1"}, Value: "1"},
			{Opcode: LiteralOp, Dst: []Reg{"r2"}, Value: "0"},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"r3"}, Src: []Reg{"r1", "r2"}},
			{Opcode: LiteralOp, Dst: []Reg{"r4"}, Value: "2"},
			{Opcode: LiteralOp, Dst: []Reg{"r5"}, Value: "0"},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"r6"}, Src: []Reg{"r4", "r5"}},
			{Opcode: LiteralOp, Dst: []Reg{"r7"}, Value: "3"},
			{Opcode: LiteralOp, Dst: []Reg{"r8"}, Value: "0"},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"r9"}, Src: []Reg{"r7", "r8"}},
			{Opcode: LiteralOp, Dst: []Reg{"r10"}, Value: "0"},
			{Opcode: BinOp, Variant: "-", Dst: []Reg{"r11"}, Src: []Reg{"r10", "r6"}},
			{Opcode: LiteralOp, Dst: []Reg{"r12"}, Value: "4"},
			{Opcode: BinOp, Variant: "*", Dst: []Reg{"r13"}, Src: []Reg{"r12", "r3"}},
			{Opcode: BinOp, Variant: "*", Dst: []Reg{"r14"}, Src: []Reg{"r13", "r9"}},
			{Opcode: BinOp, Variant: "*", Dst: []Reg{"r15"}, Src: []Reg{"r6", "r6"}},
			{Opcode: BinOp, Variant: "-", Dst: []Reg{"r16"}, Src: []Reg{"r14", "r15"}},
			{Opcode: LiteralOp, Dst: []Reg{"r17"}, Value: "2"},
			{Opcode: BinOp, Variant: "*", Dst: []Reg{"r18"}, Src: []Reg{"r17", "r3"}},
			{Opcode: BinOp, Variant: "/", Dst: []Reg{"r19"}, Src: []Reg{"r16", "r18"}},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"r20"}, Src: []Reg{"r11", "r19"}},
			{Opcode: ReturnOp, Dst: nil, Src: []Reg{"r20"}},
		},
	}
	fun := &Func{
		Name:   "<toplevel>",
		blocks: []*block{&entry},
	}

	b := fun.blocks[0].SelectInstructions(fun)
	if err := b.checkMachineInstructions(); err != nil {
		t.Fatal(err)
	}
	p := &asmProg{blocks: []*asmBlock{b}}
	//p.assignHomes()
	R := regalloc(p.blocks)
	// FIXME: this is nondeterministic somehow
	fmt.Println(R)

	// None of the registers which are live across the division should be assigned to rdx
	rdx := 1 // from the registers array
	if r, ok := R[asmArg{Var: "r11"}]; !ok {
		t.Errorf("variable r11 not assigned any register (want any register except rdx)")
	} else if r == rdx {
		t.Errorf("variable r11 assigned to rdx, want any other register")
	}

	// Also, the second argument to the division shouldn't be assiged to edx either,
	// since we have to cdq first.
	if r, ok := R[asmArg{Var: "r18"}]; !ok {
		t.Errorf("variable r18 not assigned any register (want any register except rdx)")
	} else if r == rdx {
		t.Errorf("variable r18 assigned to rdx, want any other register")
	}
}
