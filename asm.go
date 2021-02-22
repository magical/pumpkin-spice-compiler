package main

import (
	"bytes"
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
	pushq %rbp
	movq %rsp, %rbp
`

const asmEpilogue = `
	popq %rbp
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

	stacksize int
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
	// a variable name, for the passes before assignHomes
	// TODO: ugh i don't like this; it should be in the portable IR, not here
	Var string
}

func (a asmArg) String() string {
	if a.Var != "" {
		return "`" + a.Var + "`" // obviously invalid asm syntax
	} else if a.Deref {
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

// Replaces all variables (asmArg with non-empty Var) with stack references
// and sets block.stacksize.
// Assumes no shadowing
func (b *asmBlock) assignHomes() {
	// for each variable we find, we bump the stack pointer
	// the stack grows down, so the variables are above the stack pointer
	// which means we need to use positive offsets from rsp
	sp := 0
	stacksize := 0
	// we keep track of the stack location of each variable in this map
	m := make(map[string]int)
	for i, l := range b.code {
		hasVars := false
		for _, a := range l.args {
			if a.isVar() {
				if _, seen := m[a.Var]; !seen {
					// TODO: get size from type info
					m[a.Var] = sp
					sp += 8 // sizeof(int)
					stacksize += 8
				}
				hasVars = true
			}
		}
		if !hasVars {
			continue
		}
		newargs := make([]asmArg, len(l.args))
		for j := range newargs {
			if l.args[j].isVar() {
				newargs[j] = mkmem("rsp", int64(m[l.args[j].Var]))
			} else {
				newargs[j] = l.args[j]
			}
		}
		b.code[i].args = newargs
	}
	if stacksize == 0 {
		b.stacksize = 0
	} else {
		b.stacksize = stacksize + (-stacksize & 15) // align to 16 bytes
	}
}

// Adds instructions to the block to adjust the stack pointer before and after the block
// 	subq rsp, $stackframe
// 	...
// 	addq rsp, $stackframe
func (b *asmBlock) addStackFrameInstructions() {
	if b.stacksize == 0 {
		return
	}
	enter := mkinstr("subq", asmArg{Reg: "rsp"}, asmArg{Imm: int64(b.stacksize)})
	exit := mkinstr("addq", asmArg{Reg: "rsp"}, asmArg{Imm: int64(b.stacksize)})
	b.code = append(b.code, exit, exit)
	copy(b.code[1:len(b.code)], b.code[0:len(b.code)-1])
	b.code[0] = enter
}

func (a *asmArg) isVar() bool { return a.Var != "" }

// checks that all the instructions in a block are actually valid x86-64 instructions
// TODO: check instruction arguments too?
func (b *asmBlock) checkMachineInstructions() error {
	for _, l := range b.code {
		if l.tag != asmInstr {
			if l.variant != "" {
				return fmt.Errorf("invalid instruction: non-empty variant in %+v", l)
			}
		} else {
			switch l.variant {
			case "movq":
			case "addq":
			case "subq":
			case "popq":
			case "pushq":
			case "ret":
			default:
				return fmt.Errorf("invalid instruction: %s is not an x86 instruction in %+v",
					l.variant, l)
			}
		}
	}
	return nil
}

// select-instructions pass
// converts from Block to asmBlock
//
// input has had complex expressions removed
// and has no shadowed variables
// runs before assignHomes

func (b *block) SelectInstructions() *asmBlock {
	var out asmBlock
	literals := make(map[Reg]int64)
	getLiteral := func(r Reg) asmArg {
		// getLiteral converts a Reg into a asmArg
		// it returns a Imm if the Reg corresponds to an integer literal
		// and a Var otherwise
		if imm, ok := literals[r]; ok {
			return asmArg{Imm: imm}
		}
		return asmArg{Var: string(r)}
	}
	for _, l := range b.code {
		switch l.Opcode {
		case LiteralOp:
			if v, ok := l.Value.(string); !ok {
				fatalf("unsupported value in LiteralOp: %v", l)
			} else {
				if n, err := strconv.ParseInt(v, 0, 64); err != nil {
					fatalf("error parsing int literal: %v: %v", l, err)
				} else {
					literals[l.Dst[0]] = n
				}
			}
		case BinOp:
			switch l.Variant {
			case "+", "-":
				op := "addq"
				if l.Variant == "-" {
					op = "subq"
				}
				// first mov the first argument to the desination
				// and the add/subtract the second argument from it.
				// if the destination is the same as the first src register,
				// we can avoid a mov
				// TODO: for addition, check if we can flip the arguments
				if l.Dst[0] != l.Src[0] {
					out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[0])))
				}
				out.code = append(out.code, mkinstr(op, asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
			default:
				fatalf("unsupported operation %s in binop: %s", l.Variant, l)
			}
		case ReturnOp:
			out.code = append(out.code, mkinstr("movq", asmArg{Reg: "rax"}, asmArg{Var: string(l.Src[0])}))
		default:
			fatalf("unhandled op: %s", l)
		}
	}
	out.label = asmLabel(b.name)
	return &out
}

func (l *Op) String() string {
	var buf bytes.Buffer
	l.debugstr(&buf)
	return buf.String()
}

// remove complex expressions nd
