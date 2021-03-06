package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kr/pretty"
)

func main() {
	if err := main3(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main3() error {
	//const source = `let v = 1 in let w = 42 in let x = v + 7 in let y = x in let z = x + w in z - y end end end end end`
	//const source = `let x = 1+0 in let y = 2+0 in if (if x < 1 then x == 0 else x == 2 end) then let z = 2+0 in y + z end else y + 10 end end end`
	const source = `let x = 1+0 in let y = 2+0 in let z = x < y in if z or x then 42 else x and y end end end end`
	expr, err := parse(strings.NewReader(source))
	if err != nil {
		return err
	}
	printExpr(expr)
	expr = uncoverBools(expr)
	prog := lower(expr)
	print(prog)

	printCFG(prog.funcs[0].blocks)

	printAsmBlock := func(b *asmBlock) {
		var p AsmPrinter
		p.w = os.Stdout
		p.ConvertBlock(b)
	}

	fmt.Println("-----------------------------------")
	var blocks []*asmBlock
	for _, irblock := range prog.funcs[0].blocks {
		blocks = append(blocks, irblock.SelectInstructions(prog.funcs[0]))
	}
	copyCFG(blocks, prog.funcs[0])
	for _, b := range blocks {
		//fmt.Println(string(b.label) + ":")
		printAsmBlock(b)
	}
	fmt.Println("-----------------------------------")
	for _, b := range blocks {
		if err := b.checkMachineInstructions(); err != nil {
			return err
		}
	}
	p := &asmProg{blocks: blocks}
	p.assignHomes()
	p.addStackFrameInstructions()
	for _, b := range p.blocks {
		b.patchInstructions()
	}

	var pr AsmPrinter
	buf := new(bytes.Buffer)
	pr.w = buf
	pr.ConvertProg(p)
	fmt.Print(buf.String())

	return compileAsm(buf.Bytes(), "./a.out")
}

func main2() error {
	x, err := parse(os.Stdin)
	if err != nil {
		return nil
	}
	printExpr(x)
	fmt.Println("=======")
	y := cpsConvert(&VarExpr{"return"}, x)
	printExpr(y)
	return err
}

func main1() error {
	// fun test expressions:
	//
	// a+b*c(d(),)
	// (func(a) let f = func(b) 5 + a end in f(a) end end)(a)
	//

	x, err := parse(os.Stdin)
	if err != nil {
		return err
	}
	pretty.Println(x)
	printExpr(x)
	y := lower(x)
	pretty.Println(y)
	print(y)

	/*
		prog := &Prog{
			blocks: []*block{
				{
					code: []Lop{
						{Op: Linit, A: "b", K: 6},
						{Op: Linit, A: "c", K: 2},
						{Op: Ladd, A: "a", B: "b", C: "c"},
					},
				},
			},
		}
		fmt.Println(gen(prog))
	*/
	return nil
}

func printCFG(blocks []*block) {
	names := func(bl []*block) (sl []string) {
		for i := range bl {
			sl = append(sl, string(bl[i].name))
		}
		return
	}
	for _, b := range blocks {
		fmt.Println(b.name, ": ", names(b.pred), "->", b.name, "->", names(b.succ))
	}
}

type ErrorList []error

func (list ErrorList) Error() string {
	var b []byte
	for i, err := range list {
		if i > 0 {
			b = append(b, '\n')
		}
		b = append(b, []byte(err.Error())...)
	}
	return string(b)
}

func parse(r io.Reader) (Expr, error) {
	l := new(lexer)
	l.Init(r)
	yyParse(l)
	var err error
	if len(l.errors) > 0 {
		err = ErrorList(l.errors)
	}
	return l.result, err
}

// does the final compile step
// of writing the assembly to a file
// and running the assembler and linker
// to produce an executable
func compileAsm(assemblyText []byte, exeName string) error {
	tempDir, err := ioutil.TempDir("", "pscbuild")
	if err != nil {
		return err
	}
	defer func() {
		log.Println("rm -rf", tempDir)
		err := os.RemoveAll(tempDir)
		if err != nil {
			log.Println(err)
		}
	}()
	asmPath := filepath.Join(tempDir, "main._psc.s")
	err = ioutil.WriteFile(asmPath, assemblyText, 0o666)
	if err != nil {
		return err
	}
	exeDir := "."
	if exePath, _ := os.Executable(); exePath != "" {
		exeDir, _ = filepath.Split(exePath)
	}
	runtimePath := filepath.Join(exeDir, "runtime.c")
	cmd := exec.Command("cc", "-O2", "-o", exeName, asmPath, runtimePath)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("compile failed: %w", err)
	}
	return err
}
