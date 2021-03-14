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

func (pr *AsmPrinter) ConvertProg(p *asmProg) {
	io.WriteString(pr.w, asmPrologue)
	for _, b := range p.blocks {
		pr.ConvertBlock(b)
	}
	io.WriteString(pr.w, asmEpilogue)
}

func (pr *AsmPrinter) convertSingleBlockProgram(b *asmBlock) {
	io.WriteString(pr.w, asmPrologue)
	pr.ConvertBlock(b)
	io.WriteString(pr.w, asmEpilogue)
}

func (pr *AsmPrinter) ConvertBlock(b *asmBlock) {
	pr.write(".L" + string(b.label) + ":\n")
	if len(b.args) > 0 {
		fatalf("block with nonzero args: %+v", b)
	}
	for _, l := range b.code {
		switch l.tag {
		case asmInstr:
			if l.variant == "movq" && len(l.args) == 2 && l.args[0] == l.args[1] {
				continue
			}
			pr.write("\t" + l.asmInstr() + "\n")
		case asmJump:
			if l.variant != "" {
				pr.write("\tj" + l.variant + " .L" + string(l.label) + "\n")
			} else {
				pr.write("\tjmp .L" + string(l.label) + "\n")
			}
		case asmCall:
			pr.write("\tcallq " + string(l.label) + "\n")
		default:
			fatalf("unhandled op: %v", l)
		}
	}
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

type asmProg struct {
	blocks    []*asmBlock
	stacksize int
}

// An asmblock is a non-portable representation of a group of assembly instructions.
type asmBlock struct {
	label asmLabel
	args  []asmArg
	code  []asmOp
	pred  []*asmBlock
	succ  []*asmBlock
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
	asmJump // jmp label
	// j$cc label
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
		return "\u2018" + a.Var + "\u2019" // obviously invalid asm syntax
	} else if a.Deref {
		return fmt.Sprintf("%d(%%%s)", a.Imm, a.Reg)
	} else if a.Reg != "" {
		return "%" + a.Reg
	} else {
		return "$" + strconv.FormatInt(a.Imm, 10)
	}
}

// The patch instructions pass fixes up instructions like
// 	movq 0(%r1), 1(%r2)
// which have too many memory references by adding an intermediate
// instruction which loads one of the memory locations into a register.
// 	movq %rax, 1(%r2)
// 	movq 0(%r1), %rax
// We assume %rax is available as a scratch register
func patchInstructions(b *asmBlock) {
	b.patchInstructions()
}

func (b *asmBlock) patchInstructions() {
	var rax = asmArg{Reg: "rax"}
	for i := 0; i < len(b.code); i++ {
		l := b.code[i]
		switch l.tag {
		case asmInstr:
			switch l.variant {
			case "imul":
				// imul's second arg can be register, memory, or imm
				// no patch necessary
			default:
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

const useFancyAllocator = false

// Replaces all variables (asmArg with non-empty Var) with stack references
// and sets prog.stacksize.
// Assumes no shadowing
func (p *asmProg) assignHomes() {
	// for each variable we find, we bump the stack pointer
	// the stack grows down, so the variables are above the stack pointer
	// which means we need to use positive offsets from rsp
	sp := 0
	stacksize := 0
	var gethome func(string) asmArg
	if useFancyAllocator {
		// allocate registers
		R := regalloc(p.blocks)
		fmt.Println(R)
		// R maps each var used by the function to a virtual register
		// each of which needs to be mapped to a machine register or a stack location
		// we keep track of the stack location of each virtual in this map
		m := make(map[int]int)
		registers := []string{"rcx", "rdx", "rsi", "rdi", "r8", "r9"}
		gethome = func(varname string) asmArg {
			// TODO: better fallback if R is incomplete
			if r := R[asmArg{Var: varname}]; r < len(registers) {
				return asmArg{Reg: registers[r]}
			} else {
				if _, seen := m[r]; !seen {
					// spill to stack
					// TODO: get size from type info
					m[r] = sp
					sp += 8 // sizeof(int)
					stacksize += 8
				}
				return mkmem("rsp", int64(m[r]))
			}
		}
	} else {
		// we keep track of the stack location of each variable in this map
		m := make(map[string]int)
		gethome = func(varname string) asmArg {
			if _, seen := m[varname]; !seen {
				// TODO: get size from type info
				m[varname] = sp
				sp += 8 // sizeof(int)
				stacksize += 8
			}
			return mkmem("rsp", int64(m[varname]))
		}
	}
	for _, b := range p.blocks {
		for i, l := range b.code {
			if len(l.args) == 0 {
				continue
			}
			newargs := make([]asmArg, len(l.args))
			for j := range newargs {
				if l.args[j].isVar() {
					newargs[j] = gethome(l.args[j].Var)
				} else {
					newargs[j] = l.args[j]
				}
			}
			b.code[i].args = newargs
		}
	}
	if stacksize == 0 {
		p.stacksize = 0
	} else {
		p.stacksize = stacksize + (-stacksize & 15) // align to 16 bytes
	}
}

// Adds instructions to the entry and exit blocks of a program to adjust the
// stack pointer before and after the code
// 	subq rsp, $stackframe
// 	...
// 	addq rsp, $stackframe
func (p *asmProg) addStackFrameInstructions() {
	if p.stacksize == 0 {
		return
	}
	if len(p.blocks) == 0 {
		return
	}
	entry := p.blocks[0]
	exit := p.blocks[len(p.blocks)-1]
	subq := mkinstr("subq", asmArg{Reg: "rsp"}, asmArg{Imm: int64(p.stacksize)})
	addq := mkinstr("addq", asmArg{Reg: "rsp"}, asmArg{Imm: int64(p.stacksize)})
	entry.code = append([]asmOp{subq}, entry.code...)
	exit.code = append(exit.code, addq)
}

func (a *asmArg) isVar() bool { return a.Var != "" }

// checks that all the instructions in a block are actually valid x86-64 instructions
// TODO: check instruction arguments too?
func (b *asmBlock) checkMachineInstructions() error {
	for _, l := range b.code {
		if l.tag != asmInstr {
			if l.tag != asmJump && l.variant != "" {
				return fmt.Errorf("invalid instruction: non-empty variant in %+v", l)
			}
		} else {
			switch l.variant {
			case "movq":
			case "addq":
			case "subq":
			case "negq":
			case "imul":
			case "idiv":
			case "cqto":
			case "cmpq":
			case "popq":
			case "pushq":
			case "setz", "setnz", "setl", "setle", "setge", "setg":
			case "movzbq":
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

func (b *block) SelectInstructions(f *Func) *asmBlock {
	arch := sysvRegisters
	rootstack := asmArg{Reg: "r15"}
	rbase := asmArg{Reg: "r11"}
	var out asmBlock
	cc := ""
	for i, l := range b.code {
		switch l.Opcode {
		case FuncLiteralOp:
			// TODO
		case LiteralOp:
			if v, ok := l.Value.(string); ok {
				if n, err := strconv.ParseInt(v, 0, 64); err != nil {
					fatalf("error parsing int literal: %v: %v", l, err)
				} else {
					f.addLiteral(l.Dst[0], n)
				}
			} else if v, ok := l.Value.(int64); ok {
				f.addLiteral(l.Dst[0], v)
			} else {
				fatalf("unsupported value in LiteralOp: %v", l)
			}
		case BinOp:
			// X86 arithmetic instructions don't have a 3-form
			// if the destination is not the same as the first src register,
			// then first mov the first argument to the desination
			// and the add/subtract the second argument from it.
			switch l.Variant {
			case "*":
				out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[0])))
				out.code = append(out.code, mkinstr("imul", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
			case "/":
				//out.code = append(out.code, mkinstr("pushq", asmArg{Reg: "rdx"}))
				out.code = append(out.code, mkinstr("movq", asmArg{Reg: "rax"}, f.getLiteral(l.Src[0])))
				out.code = append(out.code, mkinstr("cqto"))
				out.code = append(out.code, mkinstr("idiv", f.getLiteral(l.Src[1]))) // TODO: can't be a literal
				out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, asmArg{Reg: "rax"}))
				//out.code = append(out.code, mkinstr("popq", asmArg{Reg: "rdx"}))
			case "+":
				if l.Dst[0] == l.Src[0] {
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
				} else if l.Dst[0] == l.Src[1] {
					// addition is associative, so we can flip the arguments
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[0])))
				} else {
					out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[0])))
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
				}
			case "-":
				if l.Dst[0] == l.Src[0] {
					out.code = append(out.code, mkinstr("subq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
				} else if f.getLiteral(l.Src[0]) == (asmArg{Imm: 0}) {
					// dst = 0 - src1
					// can be compiled to a negq instruction
					if l.Dst[0] == l.Src[1] {
						out.code = append(out.code, mkinstr("negq", asmArg{Var: string(l.Dst[0])}))
					} else {
						out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
						out.code = append(out.code, mkinstr("negq", asmArg{Var: string(l.Dst[0])}))
					}
				} else {
					out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[0])))
					out.code = append(out.code, mkinstr("subq", asmArg{Var: string(l.Dst[0])}, f.getLiteral(l.Src[1])))
				}
			case "eq", "ne", "<", "<=", ">=", ">":
				var cc string
				switch l.Variant {
				case "eq":
					cc = "z"
				case "ne":
					cc = "nz"
				case "<=":
					cc = "le"
				case "<":
					cc = "l"
				case ">":
					cc = "g"
				case ">=":
					cc = "ge"
				}
				out.code = append(out.code, mkinstr("cmpq", f.getLiteral(l.Src[0]), f.getLiteral(l.Src[1])))
				out.code = append(out.code, mkinstr("set"+cc, asmArg{Reg: "al"}))
				out.code = append(out.code, mkinstr("movzbq", asmArg{Var: string(l.Dst[0])}, asmArg{Reg: "al"}))

			default:
				fatalf("unsupported operation %s in binop: %s", l.Variant, l)
			}
		case CompareOp:
			if !(i+1 < len(b.code) && b.code[i+1].Opcode == BranchOp) {
				fatalf("compare must be followed by a branch: %d %s", i, l)
			}
			out.code = append(out.code, mkinstr("cmpq", f.getLiteral(l.Src[0]), f.getLiteral(l.Src[1])))
			switch l.Variant {
			case "eq":
				cc = "z"
			case "ne":
				cc = "nz"
			case "<=":
				cc = "le"
			case "<":
				cc = "l"
			case ">":
				cc = "g"
			case ">=":
				cc = "ge"
			default:
				fatalf("unsupported variant %q in compare op: %s", l.Variant, l)
			}
			_ = cc
		case BranchOp:
			if !(i-1 >= 0 && b.code[i-1].Opcode == CompareOp) {
				fatalf("branch must be preceded by compare: %d %s", i, l)
			}
			out.code = append(out.code, asmOp{tag: asmJump, variant: cc, label: asmLabel(l.Label[0])})
			out.code = append(out.code, asmOp{tag: asmJump, label: asmLabel(l.Label[1])})
		case CallOp:
			if len(l.Src) > len(arch.Args) {
				fatalf("too many arguments in call op: have %d but only %d registers: %v", len(l.Src), len(arch.Args), l.String())
			}
			for i, a := range l.Src {
				i := i + 1 // XXX
				out.code = append(out.code, mkinstr("movq", asmArg{Reg: string(arch.Args[i+1])}, f.getLiteral(a)))
			}
			// XXX uhh psc_newtuple should definitely be its own Op
			out.code = append(out.code, mkinstr("movq", asmArg{Reg: string(arch.Args[0])}, rootstack))
			out.code = append(out.code, asmOp{tag: asmCall, label: asmLabel(l.Variant)})
			out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, asmArg{Reg: "rax"}))
		case RecordSetOp:
			index := l.Value.(int64)
			out.code = append(out.code, mkinstr("movq", rbase, asmArg{Var: string(l.Src[0])})) // tuple address
			out.code = append(out.code, mkinstr("movq", mkmem(rbase.Reg, index*8), f.getLiteral(l.Src[1])))
		case RecordGetOp:
			index := l.Value.(int64)
			out.code = append(out.code, mkinstr("movq", rbase, asmArg{Var: string(l.Src[0])})) // tuple address
			out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, mkmem(rbase.Reg, index*8)))
		case JumpOp:
			params := f.getBlockArgs(l.Label[0])
			if len(l.Src) != len(params) {
				fatalf("mismatched args in jump: %s -> %s", params, l.Src)
			}
			for i, a := range l.Src {
				out.code = append(out.code, mkinstr("movq", asmArg{Var: string(params[i])}, f.getLiteral(a)))
			}
			out.code = append(out.code, asmOp{tag: asmJump, label: asmLabel(l.Label[0])})
		case ReturnOp:
			out.code = append(out.code, mkinstr("movq", asmArg{Reg: "rax"}, f.getLiteral(l.Src[0])))
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

func (f *Func) addLiteral(r Reg, value int64) {
	if f.literals == nil {
		f.literals = make(map[Reg]int64)
	}
	f.literals[r] = value
}

// getLiteral converts a Reg into a asmArg
// it returns a Imm if the Reg corresponds to an integer literal
// and a Var otherwise
func (f *Func) getLiteral(r Reg) asmArg {
	if imm, ok := f.literals[r]; ok {
		return asmArg{Imm: imm}
	}
	return asmArg{Var: string(r)}
}

func (f *Func) getBlockArgs(label Label) []Reg {
	for _, b := range f.blocks {
		if b.name == label {
			return b.args
		}
	}
	return nil
}

func copyCFG(blocks []*asmBlock, f *Func) {
	m := map[Label]*asmBlock{}
	for _, b := range blocks {
		m[Label(b.label)] = b
	}
	for _, src := range f.blocks {
		b := m[src.name]
		b.pred = make([]*asmBlock, len(src.pred))
		for i := range src.pred {
			b.pred[i] = m[src.pred[i].name]
		}
		b.succ = make([]*asmBlock, len(src.succ))
		for i := range src.succ {
			b.succ[i] = m[src.succ[i].name]
		}
	}
}

func (a *asmArg) isReg() bool { return !a.Deref && a.Reg != "" }
