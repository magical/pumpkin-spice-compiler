package main

// reg - register allocator

import "sort"

type variable = asmArg

// Structures for the graph coloring algorithm
// used for register allocation
// colorNode is a node in the graph
type colorNode struct {
	// The variable or register this node represents
	Var variable
	// Our assigned register, or -1 if none is assigned yet
	Reg int
	// Conflict set. Nodes that this one conflicts with
	Conflict []*colorNode
	// Move set. List of vertices which are move-related to this one
	Moves []*colorNode
	// Saturation set. List of registers already assigned to neighboring nodes.
	InUse []int
	// Used to break ties
	Order int
}

type regallocParams struct {
	Registers  []string // names of registers to allocate, in order
	CallerSave []string // names of registers that are not preserved across calls
	CalleeSave []string // names of registers that are preserved across calls
	Args       []string // registers used to pass arguments to functions, in order
}

// Registers used in the SYS V 64-bit ABI calling convention
//
// https://wiki.osdev.org/System_V_ABI
// > Functions preserve the registers rbx, rsp, rbp, r12, r13, r14, and r15;
// > while rax, rdi, rsi, rdx, rcx, r8, r9, r10, r11 are scratch registers.
//
// we reserve rax and r11 as scratch registers
// and r15 as the rootstack register
var sysvRegisters = &regallocParams{
	Registers: []string{"rcx", "rdx", "rsi", "rdi", "r8", "r9"},
	// TODO: enable callee-save registers and push them in the prologue
	//Registers:  []string{"rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "rbx", "r12", "r13", "r14", "r15"},
	CalleeSave: []string{"rbx", "r12", "r13", "r14", "r15"},
	CallerSave: []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9", "r10", "r11"},
	Args:       []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"},
}

func regalloc(f []*asmBlock) map[variable]int {
	// TODO: sort blocks in topological order
	//
	// Build liveness sets
	// O(instruction * variables)
	//
	// TODO: liveness sets are really inefficent
	// they use space proportional to the number of instructions
	// and the average number of live variables at each instruction,
	// which is equal to the sum of the lifetimes of each variable,
	// or worst case O(n*n) where n = number of instructions.
	// what we really want is a bag of liveness spans for each variable,
	// which would use O(V) space,
	// and which we could then query efficiently in something like O(avg(L)) time
	// ALTERNATIVELY, use a bitset and limit to 64 variables
	var L = make(map[*asmBlock][][]variable) // should probably be a field on asmBlock
	for j := len(f) - 1; j >= 0; j-- {
		b := f[j]
		// the initial live set of a block is the union
		// of liveBefore[0] of all its successor blocks
		initialSet := make(map[variable]bool)
		for _, succBlock := range b.succ {
			for _, v := range L[succBlock][0] {
				initialSet[v] = true
			}
		}
		L[b] = b.computeLiveSets(initialSet)
	}
	// construct list of caller-save registers
	params := sysvRegisters // TODO: don't hardcode
	var callerSave []int
	for _, reg := range params.CallerSave {
		r, ok := findString(params.Registers, reg)
		if ok {
			callerSave = append(callerSave, r)
		}
	}
	// Build conflict graph
	V := []variable{}
	G := make(map[variable]*colorNode)
	for _, b := range f {
		L := L[b]
		for i := range b.code {
			// construct the node
			// TODO: ugh, this would be so much easier in the SSA blocks,
			// where we have Dst and Src slices, instead of asm blocks,
			// where we have to deal with actual machine instructions
			a := b.code[i].dest()
			if a == nil || !a.isVar() && !a.isReg() {
				continue
			}
			dst := *a
			if _, found := G[dst]; !found {
				if dst.isReg() {
					r, ok := findString(params.Registers, dst.Reg)
					if ok {
						G[dst] = &colorNode{Var: dst, Reg: r, Order: -r}
					}
				} else {
					G[dst] = &colorNode{Var: dst, Reg: -1, Order: len(V)}
					V = append(V, dst)
				}
			}
			node := G[dst]
			if node == nil && dst.isReg() {
				continue
			}
			// add neighbors
			switch b.code[i].tag {
			case asmInstr:
				if b.code[i].variant == "movq" {
					src := b.code[i].args[1]
					if src.isVar() {
						node.addMoveRelated(G[src])
						G[src].addMoveRelated(node)
					}
					for _, v := range L[i+1] {
						if dst != v && (!src.isVar() || src != v) && (!src.isReg() || src != v) && G[v] != nil {
							node.addConflict(G[v])
							G[v].addConflict(node)
						}
					}
				} else {
					for _, v := range L[i+1] {
						if dst != v && G[v] != nil {
							node.addConflict(G[v])
							G[v].addConflict(node)
						}
					}
				}
			case asmCall:
				// Variables which are live across a call
				// cannot be stored in caller-save registers,
				// since the callee will not preserve their values.
				// Therefore we have to be sure not to assign
				// a caller-save register to any variable live
				// at the time of a call.
				// We could do that by introducing a conflict between those registers
				// and the live set. OR, more expediently,
				// by adding the caller-save registers to the InUse
				// set of each variable.
				for _, v := range L[i+1] {
					other := G[v]
					for _, reg := range callerSave {
						if !other.isRegInUse(reg) {
							other.InUse = append(other.InUse, reg)
						}
					}
				}
			case asmJump:
				// TODO: treat as a mov between its args
				// and the target block's params
			default:
				fatalf("unhandled instruction in regalloc: %v", b.code[i])
			}
		}
	}
	// Initialize InUse of neighbors of preallocated registers
	for _, node := range G {
		if node.Reg == -1 {
			continue
		}
		for _, other := range node.Conflict {
			if !other.isRegInUse(node.Reg) {
				other.InUse = append(other.InUse, node.Reg)
			}
		}
	}
	// Color the graph
	var S []*colorNode
	for _, node := range G {
		if node.Reg == -1 {
			S = append(S, node)
		}
	}
	for k := 0; k < len(V); k++ {
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
	return R
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
func (b *asmBlock) computeLiveSets(initialSet map[variable]bool) (liveSets [][]variable) {
	L := make([][]variable, len(b.code)+1)
	live := initialSet
	if live == nil {
		live = make(map[variable]bool)
	}
	for k := len(b.code) - 1; k >= 0; k-- {
		if d := b.code[k].dest(); d != nil && (d.isVar() || d.isReg()) {
			delete(live, *d)
		}
		for _, s := range b.code[k].src() {
			if s.isVar() || s.isReg() {
				live[s] = true
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
		switch l.variant {
		case "idiv":
			return &asmArg{Reg: "rdx"} // and rax
		case "imul":
			return &asmArg{Reg: "rdx"} // and rax
		case "cmpq":
			// nothing
		case "cqto":
			return &asmArg{Reg: "rdx"}
		default:
			return &l.args[0]
		}
	}
	return nil
}

func (l *asmOp) src() []asmArg {
	switch l.tag {
	case asmInstr:
		switch l.variant {
		case "movq", "movzbq":
			return l.args[1:]
		case "addq", "subq", "cmpq":
			return l.args[0:]
		case "cqto":
			return nil // actually rax
		default:
			return l.args[0:]
		}
	}
	return nil
}

func findString(a []string, needle string) (int, bool) {
	for i, v := range a {
		if needle == v {
			return i, true
		}
	}
	return -1, false
}
