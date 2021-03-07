package main

import (
	"regexp"
	"strings"
	"testing"
)

var typecheckTests = []struct {
	input string
	typ   Type
}{
	{"true", BoolT{}},
	{"false", BoolT{}},
	{"1", IntT{}},
	{"2 + 2", IntT{}},
	{"2 < 1", BoolT{}},
	{"let a = 42 in a end", IntT{}},
	{"let a = 42 in a == 42 end", BoolT{}},
	{"let a = 42 in 42 == a end", BoolT{}},
	{"let a = 42 in a == a end", BoolT{}},
	{"true == false", BoolT{}},
	{"if true then 1 else 0 end", IntT{}},
	{"let a = true in let b = false in let c = true in (a or b) and (b or c) and (a or c) end end end", BoolT{}},
	{"(func inf() 1+inf() end)()", IntT{}},
}

var typecheckErrorTests = []struct {
	input string
	typ   Type
	error string
}{
	{"1 == 2 == 3", BoolT{}, "cannot compare main.BoolT and main.IntT"},
	{"true + false", IntT{}, "operands to [+] must be IntT, found .*"},
	{"true < false", BoolT{}, "operands to < must be IntT, found .*"},
	{"true == 1", BoolT{}, "cannot compare .* and .*"},
	{"46 and 2", BoolT{}, "operands to 'and' must be BoolT, found .*"},
	{"2 or 3", BoolT{}, "operands to 'or' must be BoolT, found .*"},
	{"if 1 then 42 else 0 end", IntT{}, "condition must be BoolT"},
	{"if true then 42 else false end", AnyT{}, "both branches.*must have the same type, found"},
	{"1(2)", AnyT{}, "cannot call non-function"},
}

func TestTypecheck(t *testing.T) {
	for _, tt := range typecheckTests {
		expr, err := parse(strings.NewReader(tt.input))
		if err != nil {
			t.Errorf("parse(%q) failed: %v", tt.input, err)
			continue
		}
		typ, err := typecheck2(expr)
		if !sameType(typ, tt.typ) {
			t.Errorf("typecheck(%q) = %#v, want %#v", tt.input, typ, tt.typ)
		}
		if err != nil {
			t.Errorf("typecheck(%q): unexpected error: %v", tt.input, err)
		}
	}
	for _, tt := range typecheckErrorTests {
		expr, err := parse(strings.NewReader(tt.input))
		if err != nil {
			t.Errorf("parse(%q) failed: %v", tt.input, err)
			continue
		}
		typ, err := typecheck2(expr)
		if !sameType(typ, tt.typ) {
			t.Errorf("typecheck(%q) = %#v, want %#v", tt.input, typ, tt.typ)
		}
		if err == nil {
			t.Errorf("typecheck(%q): expected an error but found none", tt.input)
		} else {
			matched, matchErr := regexp.MatchString(tt.error, err.Error())
			if matchErr != nil {
				t.Errorf("invalid tt.error (%q): %v", tt.error, matchErr)
			} else if !matched {
				t.Errorf("typecheck(%q): unexpected error: %v", tt.input, err)
				t.Errorf("typecheck(%q): expected error matching %q", tt.input, tt.error)
			}
		}
	}
}
