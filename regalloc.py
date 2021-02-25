# implements a graph coloring algorithm for register allocatioqn
#
import itertools

# input is a list of instruction
# an instruction is a tuple
# (op, arg, arg...)
#
# e.g.
#
# ("add", "x", "y")
# ("mov", "y", "y")
#
# instructions take 1 or 2 args
# the first one is the destination
# args can be either an int or a register name (arbitrary string)
#
# "mov" has special meaning to the allocator
#
# ---
#
# output is a list of instructions
#
# all str args will be replaced with a str like "r1"
# the number counts up from 1 and there is no upper limit,
# though we do try to minimize the number of unique names needed

def maybeint(x):
    if x.isdigit():
        return int(x)
    else:
        return x

input = """
mov x 1
mov y 2
add x y
mov z 3
mov t x
add t x
add z t
mov y x
add y z
"""

input = """
mov v 1
mov w 42
mov x v
add x 7
mov y x
mov z x
add z w
mov t y
neg t
mov %rax z
add %rax t
"""

input = [tuple(map(maybeint, line.strip().split())) for line in input.strip("\n").splitlines()]

def regalloc(list_of_instructions):
    # THE ALGORITHM
    #
    # 1.
    #
    # first we get a list of all the variables in the program.
    # we're going to use these to build a graph later
    #
    variables = set()
    for instruction in list_of_instructions:
        variables.update(filter(isvar, instruction[1:]))
    #
    # 2.
    #
    # next, we compute the liveness of each value at to every instruction
    #
    # note: _values_, not variables
    # if we modify a variable then the old value is gone, it can no longer be used
    # so we don't consider it to be live
    #
    # a value is live at instruction k if it is _read_ by any instruction after that point
    #
    # start by initializing the liveness of every instruction to the empty set
    # also add an extra one at the end
    #
    live_before = [set() for _ in list_of_instructions] + [set()]
    #
    # now build up the liveness by working backwards from the last instruction.
    #
    # the liveness set starts out empty.
    # at each instructon,
    # if we read from a variable then its value must be live before that point (by definition)
    # if we write to a variable then its value cannot be live before that point
    # (because we just created it)
    #
    # if we do both, then reading wins because we're going backwards,
    # and instructions read before writing
    #
    # special case: mov
    #
    live = set()
    for i, instruction in reversed(list(enumerate(list_of_instructions))):
        if len(instruction) >= 2:
            writes = instruction[1:2]
            reads = instruction[1:]
            if instruction[0] == "mov":
                reads = instruction[2:]
            live -= set(filter(isvar, writes))
            live |= set(filter(isvar, reads))
        live_before[i] = set(live) # copy
    for i, (live, instruction) in enumerate(zip(live_before, list_of_instructions)):
        print(i, "{" + " ".join(sorted(live)) + "}")
        print("\t", " ".join(map(str, instruction)))
    print()
    #
    # 3.
    #
    # build a graph where our variables are the vertices
    # and there is an edge between two vertices if the
    # two variables are ever live at the same time
    #
    # we can do this efficiently by walking over the list
    # of instructions and adding an edge between the written
    # variable and every variable which is live after that instruction
    #
    # our graph will be stored as a dict which maps a vertex
    # to its adjacent vertices.
    # because the graph is unidirectonal,
    # we will have to remember to add edges in both directions
    #
    # special case: if the instruction is a mov instruction,
    # then we know that both variables will have the same
    # value after the instruction (because that's what a mov does)
    # so the variables don't actually conflict here!
    # even if both values are live after the call (remember that the
    # live set tracks live _values_, not variables) we don't
    # have to mark them as conflicting with each other because
    # both variables hold the same value, so they can share a register.
    #
    G = {v: set() for v in variables}
    #
    for i, instruction in enumerate(list_of_instructions):
        if len(instruction) >= 2 and isvar(instruction[1]):
            this_vertex = instruction[1]
            extra_vertex = None
            if instruction[0] == "mov" and isvar(instruction[2]):
                extra_vertex = instruction[2]
            for other_vertex in live_before[i+1]:
                if this_vertex != other_vertex and extra_vertex != other_vertex:
                    G[this_vertex].add(other_vertex)
                    G[other_vertex].add(this_vertex)
    print("graph")
    for v in sorted(variables):
        print(v, "->", "{" + " ".join(sorted(G[v])) + "}")
    #
    # 4.
    #
    # construct move-related graph
    #
    # this keeps track of which variables are related by a move instruction.
    # it is used for move biasing, explained in the next step.
    #
    M = {v: set() for v in variables}
    for i, instruction in enumerate(list_of_instructions):
        if len(instruction) >= 3 and instruction[0] == "mov":
            this_vertex = instruction[1]
            other_vertex = instruction[2]
            if isvar(this_vertex) and isvar(other_vertex) and this_vertex != other_vertex:
                M[this_vertex].add(other_vertex)
                M[other_vertex].add(this_vertex)
    #
    # 5.
    #
    # time to color this graph!
    # how do we do that?
    #
    # the numbered registers will be our colors
    #
    # we'll use a greedy algorithm:
    # i.   choose the _most constrained_ variable
    #      (the one which conflicts with the most _already colored_ variables)
    #      (break ties deterministicly)
    # ii.  assign a register to that variable
    # so far so good
    # iii. pick the next most constrained variable
    # iv.  assign it the _lowest register_ which _isn't_ in use by
    #      any of its neighbors
    # v.   repeat until all variables are assigned a register
    #
    # optimization: move biasing.
    # when picking the register for a vertex,
    # try to use the same register number as a move-related variable
    # that has already been assigned a register, if possible
    #
    S = {v: set() for v in variables}
    R = {}
    for _ in variables:
        max_saturation = max(len(S[v]) for v in S if v not in R)
        v = min(v for v in variables if v not in R if len(S[v]) == max_saturation)
        in_use = set(R[x] for x in G[v] if x in R)
        assert in_use == S[v]
        for related_vertex in sorted(M[v]):
            if related_vertex in R and R[related_vertex] not in in_use:
                R[v] = R[related_vertex]
                break
        else:
            for r in itertools.count():
                if r not in in_use:
                    R[v] = r
                    break
        for other_vertex in G[v]:
            S[other_vertex].add(R[v])
        print("R", v, "<-", R[v])
    print("R", R)
    #
    # we're done!
    #
    # print out the program with all the variables replaced by
    # their assigned registers, so we can see the fruits of our labors
    #
    for i, instruction in enumerate(list_of_instructions):
        new_instruction = list(instruction[0:1]) + ["r{}".format(R[x]) if x in R else x for x in instruction[1:]]
        if len(new_instruction) == 3 and instruction[0] == "mov" and new_instruction[1] == new_instruction[2]:
            # ignore mov a, a
            continue
        print(i, " ".join(str(x) for x in new_instruction))


def isvar(x):
    return isinstance(x, str) and not x.startswith("%")

regalloc(input)
