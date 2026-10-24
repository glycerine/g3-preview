package cmd_context

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func logicPin(t *testing.T, m *ast.ASTManager, h ast.ASTHandle, err error) ast.ASTHandle {
	t.Helper()
	contextMust(t, err)
	r, err := m.NewRef(h)
	contextMust(t, err)
	t.Cleanup(func() { contextMust(t, m.DelRef(&r)) })
	return h
}

func TestLogicCheckerSortPoliciesAndBorrowedLifetime(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	b, err := m.MkBoolSort()
	contextMust(t, err)
	i, err := c.LookupSort("Int")
	contextMust(t, err)
	r, err := c.LookupSort("Real")
	contextMust(t, err)
	bv, err := m.BitVectorSort(8)
	contextMust(t, err)
	name, err := m.Symbols.Intern("raw-U")
	contextMust(t, err)
	raw, e := m.MkSort(name, ast.ASTSortInfo{})
	raw = logicPin(t, m, raw, e)
	user, err := c.DeclareSort("user-U")
	contextMust(t, err)
	array, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bv}), bv)
	array = logicPin(t, m, array, e)
	multi, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{bv, bv}), bv)
	multi = logicPin(t, m, multi, e)
	badArray, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{b}), b)
	badArray = logicPin(t, m, badArray, e)
	nested, e := m.MkArraySort(ast.SliceArguments([]ast.ASTHandle{i}), bv)
	nested = logicPin(t, m, nested, e)
	for _, tc := range []struct {
		logic    string
		sort     ast.ASTHandle
		function bool
		want     string
	}{
		{"QF_UF", i, false, "integers"}, {"QF_LIA", r, false, "reals"},
		{"QF_LIA", bv, false, "bitvectors"}, {"QF_LIA", raw, false, "uninterpreted sorts"},
		{"QF_LIA", user, false, ""}, // Source user-sort family is not null_family_id.
		{"QF_UF", raw, false, ""}, {"QF_LIA", b, true, "uninterpreted functions"},
		{"QF_UF", b, true, ""}, {"QF_ABV", array, false, ""},
		{"QF_ABV", multi, false, "unidimensional arrays"},
		{"QF_ABV", badArray, false, "arrays from bitvectors to bitvectors"},
		{"QF_BV", array, false, "does not support arrays"},
		{"QF_AX", nested, false, ""}, // General arrays do not recursively check sorts.
		{"QF_S", i, false, ""}, {"QF_S", r, false, ""},
		{"QF_FD", bv, false, ""}, {"QF_FD", r, false, "reals"},
		{"ALL", raw, true, ""}, {"LIA", r, true, ""}, // No source check_logic profile.
	} {
		t.Run(tc.logic+"/"+tc.want, func(t *testing.T) {
			var checker LogicChecker
			contextMust(t, checker.SetLogic(m, tc.logic))
			var domain []ast.ASTHandle
			if tc.function {
				domain = []ast.ASTHandle{tc.sort}
			}
			d, e := m.MkFuncDecl(usymbol.Numerical(900), domain, tc.sort, ast.ASTFuncDeclInfo{})
			d = logicPin(t, m, d, e)
			err := checker.CheckDeclaration(d)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatal("policy", err, tc.want)
			}
			if err != nil && checker.LastError() != err.Error() {
				t.Fatal("last error lost", checker.LastError(), err)
			}
			checker.Reset()
			if checker.LastError() != "" {
				t.Fatal("reset retained diagnostic")
			}
			contextMust(t, checker.CheckExpression(ast.ASTHandle{}))
		})
	}
}

func TestLogicCheckerPostorderDifferenceFormsAndFailureAtomicity(t *testing.T) {
	c := contextFixture(t)
	contextMust(t, c.SetLogic("QF_IDL"))
	m, err := c.Manager()
	contextMust(t, err)
	i, err := c.LookupSort("Int")
	contextMust(t, err)
	for _, name := range []string{"x", "y"} {
		_, err = c.DeclareFunction(name, nil, i)
		contextMust(t, err)
	}
	x, err := c.MakeApp("x", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	y, err := c.MakeApp("y", nil, nil, ast.ASTHandle{})
	contextMust(t, err)
	zero, e := m.ArithmeticIntegerNumeralText("0")
	zero = logicPin(t, m, zero, e)
	one, e := m.ArithmeticIntegerNumeralText("1")
	one = logicPin(t, m, one, e)
	arith, err := m.ArithmeticFamilyID()
	contextMust(t, err)
	app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle {
		h, e := m.MkFamilyApp(arith, op, nil, args)
		return logicPin(t, m, h, e)
	}
	sum := app(6, x, y)
	difference := app(7, x, y)
	xx := app(6, x, x)
	yy := app(6, y, y)
	xOffset := app(6, one, app(7, x, one))
	var checker LogicChecker
	contextMust(t, checker.SetLogic(m, "QF_IDL"))
	for _, term := range []ast.ASTHandle{difference, app(7, xx, yy), xOffset} {
		contextMust(t, checker.CheckExpression(app(2, term, zero)))
	}
	bad := app(2, sum, zero)
	if err = checker.CheckExpression(bad); err == nil || err.Error() != "logic only supports difference arithmetic" {
		t.Fatal(err)
	}
	last := checker.LastError()
	contextMust(t, checker.CheckExpression(app(2, difference, zero)))
	if checker.LastError() != last {
		t.Fatal("success erased source last_error")
	}
	distinct, e := m.MkDistinct([]ast.ASTHandle{sum, zero})
	distinct = logicPin(t, m, distinct, e)
	contextMust(t, checker.CheckExpression(distinct)) // Boolean result: source check_diff_args does not inspect children.
	status, err := c.CheckSat(nil)
	contextMust(t, err)
	if status != lbool.True {
		t.Fatal("initial empty result", status)
	}
	if err = c.Assert(bad, "rejected"); err == nil {
		t.Fatal("invalid difference assertion accepted")
	}
	if len(c.assertions) != 0 {
		t.Fatal("rejected assertion was published")
	}
	if !c.checked || c.status != lbool.True {
		t.Fatal("refusal invalidated the preceding check result")
	}
	contextMust(t, c.Assert(app(2, difference, zero), ""))
	if len(c.assertions) != 1 {
		t.Fatal("valid assertion lost after refusal")
	}
	if _, err = c.DeclareFunction("wrong-sort", nil, mustLogicReal(t, c)); err == nil {
		t.Fatal("invalid declaration accepted")
	}
	if _, ok := c.functions["wrong-sort"]; ok {
		t.Fatal("rejected declaration was published")
	}
}

func mustLogicReal(t *testing.T, c *Context) ast.ASTHandle {
	t.Helper()
	s, e := c.LookupSort("Real")
	contextMust(t, e)
	return s
}

type logicClosedWriter struct{}

func (logicClosedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestLogicCheckerDifferenceTraceAndClosedStream(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	i, err := c.LookupSort("Int")
	contextMust(t, err)
	x, e := m.MkConst(usymbol.Numerical(42), i)
	x = logicPin(t, m, x, e)
	arith, err := m.ArithmeticFamilyID()
	contextMust(t, err)
	sum, e := m.MkFamilyApp(arith, 6, nil, []ast.ASTHandle{x, x})
	sum = logicPin(t, m, sum, e)
	zero, e := m.ArithmeticIntegerNumeralText("0")
	zero = logicPin(t, m, zero, e)
	bad, e := m.MkFamilyApp(arith, 2, nil, []ast.ASTHandle{sum, zero})
	bad = logicPin(t, m, bad, e)
	var checker LogicChecker
	contextMust(t, checker.SetLogic(m, "QF_IDL"))
	v := logicCheckInvocation{checker: &checker, apps: make(map[ast.ASTHandle]logicCheckApp)}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("check_logic")
	for _, trace := range []*diagnostic.TraceTags{tags, diagnostic.NewTraceTagsWithWriter(logicClosedWriter{})} {
		trace.Enable("check_logic")
		if err := v.nonDiffTrace(bad, trace); err == nil || err.Error() != "logic only supports difference arithmetic" || checker.LastError() != err.Error() {
			t.Fatal("trace changed validation failure", err, checker.LastError())
		}
	}
	if tags.ShouldTrace("check_logic") {
		if !strings.Contains(output.String(), "[check_logic] fail_non_diff src/cmd_context/check_logic.cpp:309") || !strings.Contains(output.String(), "(<= (+ k!42 k!42) 0)") {
			t.Fatal("missing source diagnostic trace", output.String())
		}
	} else if output.Len() != 0 {
		t.Fatal("erased trace emitted output", output.String())
	}
}

func TestLogicCheckerTheoryFamiliesAndQuantifierChildren(t *testing.T) {
	c := contextFixture(t)
	m, err := c.Manager()
	contextMust(t, err)
	b, err := m.MkBoolSort()
	contextMust(t, err)
	i, err := c.LookupSort("Int")
	contextMust(t, err)
	p, e := m.MkConst(usymbol.Numerical(1), b)
	p = logicPin(t, m, p, e)
	q, e := m.MkConst(usymbol.Numerical(2), b)
	q = logicPin(t, m, q, e)
	pb, e := m.MkPseudoBooleanAtMost([]ast.ASTHandle{p, q}, 1)
	pb = logicPin(t, m, pb, e)
	fps, err := m.FloatingPointSort(8, 24)
	contextMust(t, err)
	nan, e := m.MkFPANaNOfSort(fps)
	nan = logicPin(t, m, nan, e)
	str, e := m.MkString("abc")
	str = logicPin(t, m, str, e)
	contextMust(t, c.DeclareDatatypes([]ast.DatatypeDefinition{{Name: usymbol.Numerical(300), Constructors: []ast.DatatypeConstructorDefinition{{Name: usymbol.Numerical(301), Recognizer: usymbol.Numerical(302)}}}}))
	dtSort := c.sortTrail[len(c.sortTrail)-1].handle
	constructors, err := m.DatatypeConstructors(dtSort)
	contextMust(t, err)
	dt, e := m.MkApp(constructors[0], nil)
	dt = logicPin(t, m, dt, e)
	for _, tc := range []struct {
		logic string
		term  ast.ASTHandle
		want  string
	}{
		{"QF_UF", nan, "logic does not support theory fpa"},
		{"QF_UF", pb, "logic does not support theory pb"},
		{"QF_FD", pb, ""}, {"QF_UF", str, ""},
		{"QF_UF", dt, "logic does not support theory datatype"}, {"QF_DT", dt, ""},
	} {
		var checker LogicChecker
		contextMust(t, checker.SetLogic(m, tc.logic))
		err := checker.CheckExpression(tc.term)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || err.Error() != tc.want) {
			t.Fatal(tc.logic, tc.want, err)
		}
	}
	truth, e := m.MkTrue()
	truth = logicPin(t, m, truth, e)
	pattern, e := m.MkPattern([]ast.ASTHandle{nan})
	pattern = logicPin(t, m, pattern, e)
	for _, tc := range []struct {
		body     ast.ASTHandle
		patterns []ast.ASTHandle
		want     string
	}{
		{truth, []ast.ASTHandle{pattern}, "logic does not support theory fpa"},
		{pb, []ast.ASTHandle{pattern}, "logic does not support theory pb"},
		{truth, nil, "logic does not support theory pb"},
	} {
		var noPatterns []ast.ASTHandle
		if len(tc.patterns) == 0 {
			noPatterns = []ast.ASTHandle{pb}
		}
		h, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{i}, Names: []usymbol.Symbol{usymbol.Numerical(0)}, Body: tc.body, Patterns: tc.patterns, NoPatterns: noPatterns})
		h = logicPin(t, m, h, e)
		var checker LogicChecker
		contextMust(t, checker.SetLogic(m, "AUFLIA"))
		if err := checker.CheckExpression(h); err == nil || err.Error() != tc.want {
			t.Fatal("body/pattern/no-pattern source order", err, tc.want)
		}
	}
}
