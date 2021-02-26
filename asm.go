package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
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

func (pr *AsmPrinter) ConvertProg(b *asmBlock) {
	// TODO: multiple blocks
	io.WriteString(pr.w, asmPrologue)
	pr.ConvertBlock(b)
	io.WriteString(pr.w, asmEpilogue)
}

func (pr *AsmPrinter) ConvertBlock(b *asmBlock) {
	pr.write("." + string(b.label) + ":\n")
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
			pr.write("\tjmp ." + string(l.label) + "\n")
		case asmCall:
			pr.write("\tcallq " + string(l.label) + "\n")
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

const useFancyAllocator = true

// Replaces all variables (asmArg with non-empty Var) with stack references
// and sets block.stacksize.
// Assumes no shadowing
func (b *asmBlock) assignHomes() {
	// for each variable we find, we bump the stack pointer
	// the stack grows down, so the variables are above the stack pointer
	// which means we need to use positive offsets from rsp
	sp := 0
	stacksize := 0
	// allocate registers
	R := regalloc([]*asmBlock{b})
	fmt.Println(R)
	if useFancyAllocator {
		// we keep track of the stack location of each virtual in this map
		m := make(map[int]int)
		registers := []string{"rcx", "rdx", "rsi", "rdi", "r8", "r9"}
		for _, r := range R {
			if _, seen := m[r]; !seen {
				if r >= len(registers) {
					// TODO: get size from type info
					m[r] = sp
					sp += 8 // sizeof(int)
					stacksize += 8
				}
			}
		}
		for i, l := range b.code {
			hasVars := false
			for _, a := range l.args {
				if a.isVar() {
					hasVars = true
				}
			}
			if !hasVars {
				continue
			}
			newargs := make([]asmArg, len(l.args))
			for j := range newargs {
				if l.args[j].isVar() {
					if r := R[l.args[j].Var]; r < len(registers) {
						newargs[j] = asmArg{Reg: registers[r]}
					} else {
						newargs[j] = mkmem("rsp", int64(m[r]))
					}
				} else {
					newargs[j] = l.args[j]
				}
			}
			b.code[i].args = newargs
		}

	} else {
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
			case "negq":
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
			// X86 arithmetic instructions don't have a 3-form
			// if the destination is not the same as the first src register,
			// then first mov the first argument to the desination
			// and the add/subtract the second argument from it.
			switch l.Variant {
			case "+":
				if l.Dst[0] == l.Src[0] {
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
				} else if l.Dst[0] == l.Src[1] {
					// addition is associative, so we can flip the arguments
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[0])))
				} else {
					out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[0])))
					out.code = append(out.code, mkinstr("addq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
				}
			case "-":
				if l.Dst[0] == l.Src[0] {
					out.code = append(out.code, mkinstr("subq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
				} else if getLiteral(l.Src[0]) == (asmArg{Imm: 0}) {
					// dst = 0 - src1
					// can be compiled to a negq instruction
					if l.Dst[0] == l.Src[1] {
						out.code = append(out.code, mkinstr("negq", asmArg{Var: string(l.Dst[0])}))
					} else {
						out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
						out.code = append(out.code, mkinstr("negq", asmArg{Var: string(l.Dst[0])}))
					}
				} else {
					out.code = append(out.code, mkinstr("movq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[0])))
					out.code = append(out.code, mkinstr("subq", asmArg{Var: string(l.Dst[0])}, getLiteral(l.Src[1])))
				}
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

// Structures for the graph coloring algorithm
// used for register allocation
// colorNode is a node in the graph
type colorNode struct {
	// The variable this node represents
	Var string
	// Our assigned register, or -1 if none is assigned yet
	Reg int
	// Conflict set. Nodes that this one conflicts with
	Conflict []*colorNode
	// Move set. List of vertices which are move-related to this one
	Moves         []*colorNode
	isMoveRelated bool
	// Saturation set. List of registers already assigned to neighboring nodes.
	InUse []int
	// Used to break ties
	Order int
}

func regalloc(f []*asmBlock) map[string]int {
	type variable = string
	// TODO: sort blocks in topological order
	/*
			V := []variable{}
			G := make(map[variable]*colorNode)
		for _, b := range f {
			for i := range b.code {
				// construct the node
				// TODO: ugh, this would be so much easier in the SSA blocks,
				// where we have Dst and Src slices, instead of asm blocks,
				// where we have to deal with actual machine instructions
				a := b.code[i].dest()
				if a == nil || !isVar(*dst) {
					continue
				}
				dst := a.Var
				if _, found := G[dst]; !found {
					G[dst] = &colorNode{Var: dst, R: -1, Order: len(variables)}
					V = append(V, dst)
				}
				node := G[dst]
				// add neighbors
				src := b.code[i].readvars()
				for _, other := range src {
				}
			}
		}
	*/
	// Build liveness sets
	// O(instruction * variables)
	var L [][]variable // should probably be a field on asmBlock
	for j := len(f) - 1; j >= 0; j-- {
		b := f[j]
		// TODO: compute initial set as the union of liveBefore[0] of all successor blocks
		L = b.computeLiveSets(nil)
	}
	// Build conflict graph
	for _, b := range f {
		V := []variable{}
		G := make(map[variable]*colorNode)
		for i := range b.code {
			// construct the node
			// TODO: ugh, this would be so much easier in the SSA blocks,
			// where we have Dst and Src slices, instead of asm blocks,
			// where we have to deal with actual machine instructions
			a := b.code[i].dest()
			if a == nil || !a.isVar() {
				continue
			}
			dst := a.Var
			if _, found := G[dst]; !found {
				G[dst] = &colorNode{Var: dst, Reg: -1, Order: len(V)}
				V = append(V, dst)
			}
			node := G[dst]
			// add neighbors
			switch b.code[i].tag {
			case asmInstr:
				if b.code[i].variant == "movq" {
					src := b.code[i].args[1]
					if src.isVar() {
						node.addMoveRelated(G[src.Var])
						G[src.Var].addMoveRelated(node)
					}
					for _, v := range L[i+1] {
						if dst != v && (!src.isVar() || src.Var != v) {
							node.addConflict(G[v])
							G[v].addConflict(node)
						}
					}
				} else {
					for _, v := range L[i+1] {
						if dst != v {
							node.addConflict(G[v])
							G[v].addConflict(node)
						}
					}
				}
			case asmCall:
				// TODO: add conflict between live variables
				// and caller-save registers
				// HOLD Up wait does that mean i need regisiters
				// in my variable graph??
			}
		}
		// Color the graph
		var S []*colorNode
		for _, node := range G {
			S = append(S, node)
		}
		for k := 0; k < len(G); k++ {
			sort.Slice(S, func(i, j int) bool {
				if len(S[i].InUse) < len(S[j].InUse) {
					return true
				}
				if len(S[i].InUse) > len(S[j].InUse) {
					return false
				}
				if !S[i].hasBias() && S[j].hasBias() {
					return true
				}
				if S[i].hasBias() && !S[j].hasBias() {
					return false
				}
				return S[i].Order >= S[j].Order
			})
			var node *colorNode
			node, S = S[len(S)-1], S[:len(S)-1] // pop
			for r := 0; ; r++ {
				if !node.isRegInUse(r) {
					node.Reg = r
					break
				}
			}
			for _, other := range node.Conflict {
				if !other.isRegInUse(node.Reg) {
					other.InUse = append(other.InUse, node.Reg)
				}
			}
		}
		// Compute register map
		R := make(map[variable]int, len(G))
		for v := range G {
			R[v] = G[v].Reg
		}
		return R // XXX
	}
	//return R
	panic("unreachable")
}

func (n *colorNode) addConflict(other *colorNode) {
	for i := range n.Conflict {
		if n.Conflict[i] == other {
			return
		}
	}
	n.Conflict = append(n.Conflict, other)
}

func (n *colorNode) addMoveRelated(other *colorNode) {
	for i := range n.Moves {
		if n.Moves[i] == other {
			return
		}
	}
	n.Moves = append(n.Moves, other)
}

func (n *colorNode) hasBias() bool {
	for _, other := range n.Moves {
		if other.Reg >= 0 && !n.isRegInUse(other.Reg) {
			return true
		}
	}
	return false
}
func (n *colorNode) isRegInUse(r int) bool {
	for _, s := range n.InUse {
		if r == s {
			return true
		}
	}
	return false
}

// if initialSet is provided, it will be used as the initial liveAfter set
// and will be modified during the computation. after returning, it will be
// the liveBefore set of the first instruction in the block.
// returns a list of live variables before each instruction in the block
func (b *asmBlock) computeLiveSets(initialSet map[string]bool) (liveSets [][]string) {
	type variable = string
	L := make([][]variable, len(b.code)+1)
	live := initialSet
	if live == nil {
		live = make(map[variable]bool)
	}
	for k := len(b.code) - 1; k >= 0; k-- {
		if d := b.code[k].dest(); d != nil && d.isVar() {
			live[d.Var] = false
		}
		for _, s := range b.code[k].src() {
			if s.isVar() {
				live[s.Var] = true
			}
		}
		for v, ok := range live {
			if ok {
				L[k] = append(L[k], v) // TODO: sort?
			}
		}
	}
	return L
}

func (l *asmOp) dest() *asmArg {
	switch l.tag {
	case asmInstr:
		if l.variant != "cmpq" {
			return &l.args[0]
		}
	}
	return nil
}
func (l *asmOp) src() []asmArg {
	switch l.tag {
	case asmInstr:
		switch l.variant {
		case "movq":
			return l.args[1:]
		case "addq":
			return l.args[0:]
		case "subq":
			return l.args[0:]
		default:
			return l.args[0:]
		}
	}
	return nil
}
