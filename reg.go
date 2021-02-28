package main

// reg - register allocator

import "sort"

type variable = string

// Structures for the graph coloring algorithm
// used for register allocation
// colorNode is a node in the graph
type colorNode struct {
	// The variable this node represents
	Var variable
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

func regalloc(f []*asmBlock) map[variable]int {
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
			case asmJump:
				// TODO: treat as a mov between its args
				// and the target block's params
			default:
				fatalf("unhandled instruction in regalloc: %v", b.code[i])
			}
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
		if d := b.code[k].dest(); d != nil && d.isVar() {
			delete(live, d.Var)
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
		case "movq", "movzbq":
			return l.args[1:]
		case "addq", "subq", "cmpq":
			return l.args[0:]
		default:
			return l.args[0:]
		}
	}
	return nil
}
