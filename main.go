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

	"github.com/kr/pretty"
)

func main() {
	if err := main3(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main3() error {
	printAsmBlock := func(b *asmBlock) {
		var p AsmPrinter
		p.w = os.Stdout
		p.ConvertBlock(b)
	}
	irblock := &block{
		name: "L0",
		code: []Op{
			{Opcode: LiteralOp, Dst: []Reg{"x"}, Value: "20"},
			{Opcode: LiteralOp, Dst: []Reg{"y"}, Value: "2"},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"z"}, Src: []Reg{"x", "x"}},
			{Opcode: BinOp, Variant: "+", Dst: []Reg{"w"}, Src: []Reg{"z", "y"}},
			{Opcode: ReturnOp, Src: []Reg{"w"}},
		},
	}
	printb(irblock)
	fmt.Println("-----------------------------------")
	block := irblock.SelectInstructions()
	printAsmBlock(block)
	fmt.Println("-----------------------------------")
	if err := block.checkMachineInstructions(); err != nil {
		return err
	}
	block.assignHomes()
	block.addStackFrameInstructions()
	block.patchInstructions()

	var p AsmPrinter
	buf := new(bytes.Buffer)
	p.w = buf
	p.ConvertBlock(block)
	fmt.Print(buf.String())

	err := compileAsm(buf.Bytes(), "./a.out")
	return err
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
	cmd := exec.Command("cc", "-o", exeName, asmPath, runtimePath, "-masm=intel")
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("compile failed: %w", err)
	}
	return err
}
