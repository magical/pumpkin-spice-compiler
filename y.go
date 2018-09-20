// Code generated by goyacc - DO NOT EDIT.

package main

import __yyfmt__ "fmt"

type yySymType struct {
	yys   int
	ident string
	num   string
}

type yyXError struct {
	state, xsym int
}

const (
	yyDefault = 57354
	yyEofCode = 57344
	yyErrCode = 57345
	kElse     = 57350
	kEnd      = 57353
	kFunc     = 57351
	kIf       = 57349
	kIn       = 57352
	kLet      = 57348
	tIdent    = 57346
	tNumber   = 57347

	yyMaxDepth = 200
	yyTabOfs   = -29
)

var (
	yyPrec = map[int]int{
		'+': 0,
		'-': 0,
		'*': 1,
		'/': 1,
		'(': 2,
	}

	yyXLAT = map[int]int{
		40:    0,  // '(' (34x)
		41:    1,  // ')' (29x)
		42:    2,  // '*' (22x)
		43:    3,  // '+' (22x)
		45:    4,  // '-' (22x)
		47:    5,  // '/' (22x)
		44:    6,  // ',' (21x)
		57353: 7,  // kEnd (18x)
		57344: 8,  // $end (17x)
		57352: 9,  // kIn (16x)
		57364: 10, // ident (14x)
		57346: 11, // tIdent (14x)
		57359: 12, // call (11x)
		57360: 13, // expr (11x)
		57363: 14, // func (11x)
		57351: 15, // kFunc (11x)
		57348: 16, // kLet (11x)
		57365: 17, // let (11x)
		57366: 18, // num (11x)
		57347: 19, // tNumber (11x)
		61:    20, // '=' (2x)
		57355: 21, // arglist0 (1x)
		57356: 22, // arglist1 (1x)
		57357: 23, // args (1x)
		57358: 24, // body (1x)
		57361: 25, // exprlist0 (1x)
		57362: 26, // exprlist1 (1x)
		57367: 27, // top (1x)
		57354: 28, // $default (0x)
		57345: 29, // error (0x)
		57350: 30, // kElse (0x)
		57349: 31, // kIf (0x)
	}

	yySymNames = []string{
		"'('",
		"')'",
		"'*'",
		"'+'",
		"'-'",
		"'/'",
		"','",
		"kEnd",
		"$end",
		"kIn",
		"ident",
		"tIdent",
		"call",
		"expr",
		"func",
		"kFunc",
		"kLet",
		"let",
		"num",
		"tNumber",
		"'='",
		"arglist0",
		"arglist1",
		"args",
		"body",
		"exprlist0",
		"exprlist1",
		"top",
		"$default",
		"error",
		"kElse",
		"kIf",
	}

	yyTokenLiteralStrings = map[int]string{}

	yyReductions = map[int]struct{ xsym, components int }{
		0:  {0, 1},
		1:  {27, 1},
		2:  {13, 1},
		3:  {17, 7},
		4:  {13, 3},
		5:  {13, 3},
		6:  {13, 3},
		7:  {13, 3},
		8:  {13, 1},
		9:  {13, 1},
		10: {13, 3},
		11: {13, 1},
		12: {14, 6},
		13: {23, 1},
		14: {24, 1},
		15: {21, 0},
		16: {21, 1},
		17: {21, 2},
		18: {22, 1},
		19: {22, 3},
		20: {13, 1},
		21: {12, 4},
		22: {25, 0},
		23: {25, 1},
		24: {25, 2},
		25: {26, 1},
		26: {26, 3},
		27: {10, 1},
		28: {18, 1},
	}

	yyXErrors = map[yyXError]string{}

	yyParseTab = [47][]uint8{
		// 0
		{36, 10: 34, 40, 39, 31, 37, 38, 33, 32, 35, 41, 27: 30},
		{8: 29},
		{57, 2: 55, 53, 54, 56, 8: 28},
		{27, 27, 27, 27, 27, 27, 27, 27, 27, 27},
		{10: 70, 40},
		// 5
		{21, 21, 21, 21, 21, 21, 21, 21, 21, 21},
		{20, 20, 20, 20, 20, 20, 20, 20, 20, 20},
		{36, 10: 34, 40, 39, 68, 37, 38, 33, 32, 35, 41},
		{18, 18, 18, 18, 18, 18, 18, 18, 18, 18},
		{42},
		// 10
		{9, 9, 9, 9, 9, 9, 9, 9, 9, 9},
		{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 20: 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1: 14, 10: 46, 40, 21: 44, 45, 43},
		{1: 49},
		// 15
		{1: 16},
		{1: 13, 6: 47},
		{1: 11, 6: 11},
		{1: 12, 10: 48, 40},
		{1: 10, 6: 10},
		// 20
		{36, 10: 34, 40, 39, 50, 37, 38, 33, 32, 35, 41, 24: 51},
		{57, 2: 55, 53, 54, 56, 7: 15},
		{7: 52},
		{17, 17, 17, 17, 17, 17, 17, 17, 17, 17},
		{36, 10: 34, 40, 39, 67, 37, 38, 33, 32, 35, 41},
		// 25
		{36, 10: 34, 40, 39, 66, 37, 38, 33, 32, 35, 41},
		{36, 10: 34, 40, 39, 65, 37, 38, 33, 32, 35, 41},
		{36, 10: 34, 40, 39, 64, 37, 38, 33, 32, 35, 41},
		{36, 7, 10: 34, 40, 39, 58, 37, 38, 33, 32, 35, 41, 25: 59, 60},
		{57, 4, 55, 53, 54, 56, 4},
		// 30
		{1: 63},
		{1: 6, 6: 61},
		{36, 5, 10: 34, 40, 39, 62, 37, 38, 33, 32, 35, 41},
		{57, 3, 55, 53, 54, 56, 3},
		{8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
		// 35
		{57, 22, 22, 22, 22, 22, 22, 22, 22, 22},
		{57, 23, 23, 23, 23, 23, 23, 23, 23, 23},
		{57, 24, 55, 24, 24, 56, 24, 24, 24, 24},
		{57, 25, 55, 25, 25, 56, 25, 25, 25, 25},
		{57, 69, 55, 53, 54, 56},
		// 40
		{19, 19, 19, 19, 19, 19, 19, 19, 19, 19},
		{20: 71},
		{36, 10: 34, 40, 39, 72, 37, 38, 33, 32, 35, 41},
		{57, 2: 55, 53, 54, 56, 9: 73},
		{36, 10: 34, 40, 39, 74, 37, 38, 33, 32, 35, 41},
		// 45
		{57, 2: 55, 53, 54, 56, 7: 75},
		{26, 26, 26, 26, 26, 26, 26, 26, 26, 26},
	}
)

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

type yyLexerEx interface {
	yyLexer
	Reduced(rule, state int, lval *yySymType) bool
}

func yySymName(c int) (s string) {
	x, ok := yyXLAT[c]
	if ok {
		return yySymNames[x]
	}

	if c < 0x7f {
		return __yyfmt__.Sprintf("%q", c)
	}

	return __yyfmt__.Sprintf("%d", c)
}

func yylex1(yylex yyLexer, lval *yySymType) (n int) {
	n = yylex.Lex(lval)
	if n <= 0 {
		n = yyEofCode
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("\nlex %s(%#x %d), lval: %+v\n", yySymName(n), n, n, lval)
	}
	return n
}

func yyParse(yylex yyLexer) int {
	const yyError = 29

	yyEx, _ := yylex.(yyLexerEx)
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, 200)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yyerrok := func() {
		if yyDebug >= 2 {
			__yyfmt__.Printf("yyerrok()\n")
		}
		Errflag = 0
	}
	_ = yyerrok
	yystate := 0
	yychar := -1
	var yyxchar int
	var yyshift int
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	if yychar < 0 {
		yylval.yys = yystate
		yychar = yylex1(yylex, &yylval)
		var ok bool
		if yyxchar, ok = yyXLAT[yychar]; !ok {
			yyxchar = len(yySymNames) // > tab width
		}
	}
	if yyDebug >= 4 {
		var a []int
		for _, v := range yyS[:yyp+1] {
			a = append(a, v.yys)
		}
		__yyfmt__.Printf("state stack %v\n", a)
	}
	row := yyParseTab[yystate]
	yyn = 0
	if yyxchar < len(row) {
		if yyn = int(row[yyxchar]); yyn != 0 {
			yyn += yyTabOfs
		}
	}
	switch {
	case yyn > 0: // shift
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		yyshift = yyn
		if yyDebug >= 2 {
			__yyfmt__.Printf("shift, and goto state %d\n", yystate)
		}
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	case yyn < 0: // reduce
	case yystate == 1: // accept
		if yyDebug >= 2 {
			__yyfmt__.Println("accept")
		}
		goto ret0
	}

	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			if yyDebug >= 1 {
				__yyfmt__.Printf("no action for %s in state %d\n", yySymName(yychar), yystate)
			}
			msg, ok := yyXErrors[yyXError{yystate, yyxchar}]
			if !ok {
				msg, ok = yyXErrors[yyXError{yystate, -1}]
			}
			if !ok && yyshift != 0 {
				msg, ok = yyXErrors[yyXError{yyshift, yyxchar}]
			}
			if !ok {
				msg, ok = yyXErrors[yyXError{yyshift, -1}]
			}
			if yychar > 0 {
				ls := yyTokenLiteralStrings[yychar]
				if ls == "" {
					ls = yySymName(yychar)
				}
				if ls != "" {
					switch {
					case msg == "":
						msg = __yyfmt__.Sprintf("unexpected %s", ls)
					default:
						msg = __yyfmt__.Sprintf("unexpected %s, %s", ls, msg)
					}
				}
			}
			if msg == "" {
				msg = "syntax error"
			}
			yylex.Error(msg)
			Nerrs++
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				row := yyParseTab[yyS[yyp].yys]
				if yyError < len(row) {
					yyn = int(row[yyError]) + yyTabOfs
					if yyn > 0 { // hit
						if yyDebug >= 2 {
							__yyfmt__.Printf("error recovery found error shift in state %d\n", yyS[yyp].yys)
						}
						yystate = yyn /* simulate a shift of "error" */
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery failed\n")
			}
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yySymName(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}

			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	r := -yyn
	x0 := yyReductions[r]
	x, n := x0.xsym, x0.components
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= n
	if yyp+1 >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	exState := yystate
	yystate = int(yyParseTab[yyS[yyp].yys][x]) + yyTabOfs
	/* reduction by production r */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce using rule %v (%s), and goto state %d\n", r, yySymNames[x], yystate)
	}

	switch r {

	}

	if yyEx != nil && yyEx.Reduced(r, exState, &yyVAL) {
		return -1
	}
	goto yystack /* stack new state and value */
}
