package smt2

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/ast/rewriter"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	usymbol "github.com/glycerine/g3/pkg/util/symbol"
)

func localLookupManager(t *testing.T) *ast.ASTManager {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, err := usymbol.NewManager(usymbol.Config{Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	q, err := unumeral.NewRationalService(unumeral.RationalServiceConfig{Integers: unumeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	if err != nil {
		t.Fatal(err)
	}
	m, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Fatal(err)
		}
		if err := q.Close(); err != nil {
			t.Fatal(err)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if b.Used() != 0 {
			t.Fatal("retained lookup owner graph", b.Used())
		}
	})
	return m
}

func localTestShifter(t *testing.T, m *ast.ASTManager) *rewriter.VarShifter {
	t.Helper()
	s, err := rewriter.NewVarShifter(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return s
}

func localTestResolve(t *testing.T, m *ast.ASTManager, s *rewriter.VarShifter, env *smt2LocalEnv, name usymbol.Symbol, bindings uint32) (ast.ASTHandle, bool, error) {
	t.Helper()
	ref, found, err := env.resolveLocal(m, s, name, bindings)
	if err != nil || !found {
		return ast.ASTHandle{}, found, err
	}
	h, err := m.RefGet(ref)
	if err != nil {
		_ = m.DelRef(&ref)
		return ast.ASTHandle{}, true, err
	}
	t.Cleanup(func() {
		if err := m.DelRef(&ref); err != nil {
			t.Fatal(err)
		}
	})
	return h, true, nil
}

func TestSMT2ParserPushLocalShiftsFreeVariables(t *testing.T) {
	m := localLookupManager(t)
	shifter := localTestShifter(t, m)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	boolSort, err := m.MkBoolSort()
	must(err)
	v0, err := m.MkVar(0, boolSort)
	must(err)
	v1, err := m.MkVar(1, boolSort)
	must(err)
	term, err := m.MkAnd(v0, v1)
	must(err)
	var env smt2LocalEnv
	name := usymbol.Numerical(7)
	env.insert(name, smt2Local{term: term, level: 1})
	shifted, found, err := localTestResolve(t, m, shifter, &env, name, 2)
	must(err)
	if !found || shifted == term {
		t.Fatal("parser did not shift a deeper local reference")
	}
	for i, want := range []uint32{1, 2} {
		child, e := m.GetArg(shifted, uint32(i))
		must(e)
		index, e := m.GetVarIndex(child)
		must(e)
		if index != want {
			t.Fatal("shifted variable index", i, index, want)
		}
	}
	same, found, err := localTestResolve(t, m, shifter, &env, name, 1)
	must(err)
	if !found || same != term {
		t.Fatal("same-level local was rebuilt")
	}
	_, found, err = localTestResolve(t, m, shifter, &env, usymbol.Numerical(8), 2)
	must(err)
	if found {
		t.Fatal("missing local resolved")
	}
	_, found, err = localTestResolve(t, m, shifter, &env, name, 0)
	if !found || !errors.Is(err, errLocalBindingLevel) {
		t.Fatal("invalid local level", found, err)
	}
	atom, err := m.MkConst(usymbol.Numerical(9), boolSort)
	must(err)
	env.insert(usymbol.Numerical(10), smt2Local{term: atom, level: 1})
	ground, found, err := localTestResolve(t, m, shifter, &env, usymbol.Numerical(10), 3)
	must(err)
	if !found || ground != atom {
		t.Fatal("ground local was rebuilt")
	}
}

func TestSMT2ParserPushLocalRespectsNestedQuantifierBinders(t *testing.T) {
	m := localLookupManager(t)
	shifter := localTestShifter(t, m)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	boolSort, err := m.MkBoolSort()
	must(err)
	v0, err := m.MkVar(0, boolSort)
	must(err)
	v1, err := m.MkVar(1, boolSort)
	must(err)
	body, err := m.MkAnd(v0, v1)
	must(err)
	q, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []usymbol.Symbol{usymbol.Numerical(3)}, Body: body})
	must(err)
	var env smt2LocalEnv
	name := usymbol.Numerical(11)
	env.insert(name, smt2Local{term: q, level: 1})
	shifted, found, err := localTestResolve(t, m, shifter, &env, name, 2)
	must(err)
	if !found || shifted == q {
		t.Fatal("quantified local did not shift free variables")
	}
	newBody, err := m.GetQuantifierBody(shifted)
	must(err)
	for i, want := range []uint32{0, 2} {
		child, e := m.GetArg(newBody, uint32(i))
		must(e)
		index, e := m.GetVarIndex(child)
		must(e)
		if index != want {
			t.Fatal("quantifier binder shift", i, index, want)
		}
	}
	pattern, err := m.MkFamilyApp(2, 0, nil, []ast.ASTHandle{body})
	must(err)
	annotated, err := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{boolSort}, Names: []usymbol.Symbol{usymbol.Numerical(3)}, Body: body, Patterns: []ast.ASTHandle{pattern}, Weight: 3, QID: usymbol.Numerical(19)})
	must(err)
	env.insert(name, smt2Local{term: annotated, level: 1})
	shifted, found, err = localTestResolve(t, m, shifter, &env, name, 2)
	must(err)
	if !found || shifted == annotated {
		t.Fatal("annotated local was not shifted")
	}
	shiftedBody, err := m.GetQuantifierBody(shifted)
	must(err)
	shiftedPattern, err := m.GetPattern(shifted, 0)
	must(err)
	patternBody, err := m.GetArg(shiftedPattern, 0)
	must(err)
	if patternBody != shiftedBody {
		t.Fatal("source shifter did not reuse the shifted body in its pattern")
	}
	weight, err := m.GetQuantifierWeight(shifted)
	must(err)
	qid, _, err := m.GetQuantifierIDs(shifted)
	must(err)
	if weight != 3 || qid != usymbol.Numerical(19) {
		t.Fatal("quantifier annotations changed", weight, qid)
	}
}

func TestSMT2ParserQualifiedLocalSortAndIndexRefusals(t *testing.T) {
	m := localLookupManager(t)
	shifter := localTestShifter(t, m)
	boolSort, err := m.MkBoolSort()
	if err != nil {
		t.Fatal(err)
	}
	otherSort, err := m.MkSort(usymbol.Numerical(22), ast.ASTSortInfo{})
	if err != nil {
		t.Fatal(err)
	}
	variable, err := m.MkVar(0, boolSort)
	if err != nil {
		t.Fatal(err)
	}
	var env smt2LocalEnv
	name := usymbol.Numerical(23)
	env.insert(name, smt2Local{term: variable, level: 1})
	before := m.Stats().LiveReferences
	ref, found, err := env.resolveQualifiedLocal(m, shifter, name, 2, boolSort, true, 0)
	if err != nil || !found {
		t.Fatal("qualified local", found, err)
	}
	h, err := m.RefGet(ref)
	if err != nil {
		t.Fatal(err)
	}
	index, err := m.GetVarIndex(h)
	if err != nil || index != 1 {
		t.Fatal("qualified local shift", index, err)
	}
	if err := m.DelRef(&ref); err != nil {
		t.Fatal(err)
	}
	if m.Stats().LiveReferences != before {
		t.Fatal("qualified local retained a temporary reference")
	}
	for _, tc := range []struct {
		sort    ast.ASTHandle
		hasSort bool
		indices uint32
		want    error
	}{
		{otherSort, true, 1, errLocalQualifiedSort}, // source sort check wins
		{boolSort, true, 1, errLocalIndexed},
		{ast.ASTHandle{}, false, 1, errLocalIndexed},
	} {
		ref, found, err := env.resolveQualifiedLocal(m, shifter, name, 2, tc.sort, tc.hasSort, tc.indices)
		if !found || ref != (ast.ASTRef{}) || !errors.Is(err, tc.want) {
			t.Fatal("qualified local refusal", tc, found, ref, err)
		}
		if m.Stats().LiveReferences != before {
			t.Fatal("refused qualified local retained expression-stack reference", tc)
		}
	}
	ref, found, err = env.resolveQualifiedLocal(m, shifter, usymbol.Numerical(24), 2, otherSort, true, 1)
	if found || err != nil || ref != (ast.ASTRef{}) {
		t.Fatal("unknown qualified name reached local checks", found, ref, err)
	}
}
