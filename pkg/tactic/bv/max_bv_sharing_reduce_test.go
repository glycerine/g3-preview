package bv

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/numeral"
	uparams "github.com/glycerine/g3/pkg/util/params"
	"github.com/glycerine/g3/pkg/util/symbol"
)

func sharingMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func sharingParamsFixture(t *testing.T, m *ast.ASTManager) (*uparams.ParamsContext, *uparams.ParamsRef) {
	t.Helper()
	b := m.Budget()
	numbers, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	sharingMust(t, e)
	ctx, e := uparams.NewParamsContext(m.Symbols, numbers, uparams.ParamsConfig{Budget: b, PageEntries: 1, Text: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}})
	sharingMust(t, e)
	v, e := ctx.NewValue()
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, ctx.Del(&v)); sharingMust(t, ctx.Close()); sharingMust(t, numbers.Close()) })
	return ctx, &v
}

func TestSharingGoalPassSourceParamsAndResourceLimits(t *testing.T) {
	m, fid := sharingFixture(t)
	ctx, params := sharingParamsFixture(t, m)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(1000+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	binary, e := m.MkFamilyApp(fid, 4, nil, vars[:2])
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
	sharingMust(t, e)
	f0, e := m.MkEq(binary, vars[2])
	sharingMust(t, e)
	f1, e := m.MkEq(nary, vars[3])
	sharingMust(t, e)
	goal, e := tactic.NewGoal(m, false, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(f0))
	sharingMust(t, goal.Assert(f1))
	key := "max_args"
	sharingMust(t, ctx.SetUIntName(params, &key, 2))
	pass, e := NewSharingGoalPassWithParams(m, ctx, params)
	sharingMust(t, e)
	_, e = pass.Apply(goal)
	sharingMust(t, e)
	form, e := goal.Form(1)
	sharingMust(t, e)
	term, e := m.GetArg(form, 0)
	sharingMust(t, e)
	left, e := m.GetArg(term, 0)
	sharingMust(t, e)
	left, e = m.GetArg(left, 0)
	sharingMust(t, e)
	if left == binary {
		t.Fatal("max_args=2 did not disable greedy pair reuse")
	}
	boolSort, e := m.MkBoolSort()
	sharingMust(t, e)
	p, e := m.MkConst(symbol.Numerical(1010), boolSort)
	sharingMust(t, e)
	limited, e := tactic.NewGoal(m, false, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, limited.Close()) })
	sharingMust(t, limited.Assert(p))
	stepsKey := "max_steps"
	sharingMust(t, ctx.SetUIntName(params, &stepsKey, 0))
	sharingMust(t, pass.UpdateParams(ctx, params))
	if _, e = pass.Apply(limited); !errors.Is(e, ErrSharingMaxSteps) {
		t.Fatal("source zero-step guard", e)
	}
	n, e := limited.Size()
	sharingMust(t, e)
	if n != 1 {
		t.Fatal("step refusal changed goal", n)
	}
	sharingMust(t, ctx.SetUIntName(params, &stepsKey, ^uint32(0)))
	memoryKey := "max_memory"
	sharingMust(t, ctx.SetUIntName(params, &memoryKey, 0))
	sharingMust(t, pass.UpdateParams(ctx, params))
	if _, e = pass.Apply(limited); !errors.Is(e, ErrSharingMaxMemory) {
		t.Fatal("source zero-memory guard", e)
	}
	n, e = limited.Size()
	sharingMust(t, e)
	if n != 1 {
		t.Fatal("memory refusal changed goal", n)
	}
}

func TestSharingGoalPassConstantITEPrunesUnselectedPairCache(t *testing.T) {
	m, fid := sharingFixture(t)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(1020+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	binary, e := m.MkFamilyApp(fid, 4, nil, vars[:2])
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
	sharingMust(t, e)
	unused, e := m.MkEq(binary, vars[2])
	sharingMust(t, e)
	formula, e := m.MkEq(nary, vars[3])
	sharingMust(t, e)
	boolSort, e := m.MkBoolSort()
	sharingMust(t, e)
	selected, e := m.MkConst(symbol.Numerical(1025), boolSort)
	sharingMust(t, e)
	for _, h := range []ast.ASTHandle{unused, formula, selected} {
		sharingMust(t, m.IncRef(h))
		h := h
		t.Cleanup(func() { sharingMust(t, m.DecRef(h)) })
	}
	truth, e := m.MkTrue()
	sharingMust(t, e)
	falsity, e := m.MkFalse()
	sharingMust(t, e)
	for _, test := range []struct {
		name     string
		cond     ast.ASTHandle
		whenTrue bool
	}{
		{"true", truth, true}, {"false", falsity, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var first ast.ASTHandle
			if test.whenTrue {
				first, e = m.MkIte(test.cond, selected, unused)
			} else {
				first, e = m.MkIte(test.cond, unused, selected)
			}
			sharingMust(t, e)
			goal, e := tactic.NewGoal(m, false, true, false)
			sharingMust(t, e)
			defer func() { sharingMust(t, goal.Close()) }()
			sharingMust(t, goal.Assert(first))
			sharingMust(t, goal.Assert(formula))
			pass, e := NewSharingGoalPass(m, 128)
			sharingMust(t, e)
			_, e = pass.Apply(goal)
			sharingMust(t, e)
			firstOut, e := goal.Form(0)
			sharingMust(t, e)
			if firstOut != selected {
				t.Fatal("constant ITE did not select source branch", firstOut, selected)
			}
			out, e := goal.Form(1)
			sharingMust(t, e)
			term, e := m.GetArg(out, 0)
			sharingMust(t, e)
			left, e := m.GetArg(term, 0)
			sharingMust(t, e)
			pairFirst, e := m.GetArg(left, 0)
			sharingMust(t, e)
			if pairFirst != vars[1] {
				t.Fatal("unselected ITE branch registered a BV pair", pairFirst, vars[1])
			}
		})
	}
}

func TestSharingGoalPassReusesEarlierFormulaAndUpdatesDepth(t *testing.T) {
	m, fid := sharingFixture(t)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(950+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	binary, e := m.MkFamilyApp(fid, 4, nil, vars[:2])
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
	sharingMust(t, e)
	f0, e := m.MkEq(binary, vars[2])
	sharingMust(t, e)
	f1, e := m.MkEq(nary, vars[3])
	sharingMust(t, e)
	goal, e := tactic.NewGoal(m, false, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(f0))
	sharingMust(t, goal.Assert(f1))
	sharingMust(t, goal.SetDepth(7))
	pass, e := NewSharingGoalPass(m, 128)
	sharingMust(t, e)
	gotGoal, e := pass.Apply(goal)
	sharingMust(t, e)
	if gotGoal != goal {
		t.Fatal("source pass must publish input goal")
	}
	depth, e := goal.Depth()
	sharingMust(t, e)
	if depth != 8 {
		t.Fatal("source depth transition", depth)
	}
	form, e := goal.Form(1)
	sharingMust(t, e)
	term, e := m.GetArg(form, 0)
	sharingMust(t, e)
	left, e := m.GetArg(term, 0)
	sharingMust(t, e)
	left, e = m.GetArg(left, 0)
	sharingMust(t, e)
	if left != binary {
		t.Fatal("earlier goal formula did not guide later pair sharing", left, binary)
	}
}

func TestSharingGoalPassVisitsQuantifiedBody(t *testing.T) {
	m, fid := sharingFixture(t)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	x, e := m.MkVar(0, sort)
	sharingMust(t, e)
	y, e := m.MkConst(symbol.Numerical(970), sort)
	sharingMust(t, e)
	z, e := m.MkConst(symbol.Numerical(971), sort)
	sharingMust(t, e)
	for _, h := range []ast.ASTHandle{x, y, z} {
		sharingMust(t, m.IncRef(h))
		h := h
		t.Cleanup(func() { sharingMust(t, m.DecRef(h)) })
	}
	binary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{x, y})
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{y, z, x, z})
	sharingMust(t, e)
	f0, e := m.MkEq(binary, z)
	sharingMust(t, e)
	f1, e := m.MkEq(nary, z)
	sharingMust(t, e)
	body, e := m.MkAnd(f0, f1)
	sharingMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(972)}, Body: body})
	sharingMust(t, e)
	goal, e := tactic.NewGoal(m, false, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(q))
	pass, e := NewSharingGoalPass(m, 128)
	sharingMust(t, e)
	_, e = pass.Apply(goal)
	sharingMust(t, e)
	got, e := goal.Form(0)
	sharingMust(t, e)
	if got == q {
		t.Fatal("quantified BV body was not rewritten")
	}
	newBody, e := m.GetQuantifierBody(got)
	sharingMust(t, e)
	second, e := m.GetArg(newBody, 1)
	sharingMust(t, e)
	term, e := m.GetArg(second, 0)
	sharingMust(t, e)
	left, e := m.GetArg(term, 0)
	sharingMust(t, e)
	left, e = m.GetArg(left, 0)
	sharingMust(t, e)
	if left != binary {
		t.Fatal("quantifier body missed cached ordered pair", left, binary)
	}
}

func TestSharingGoalPassProofChainsCongruenceAndRewrite(t *testing.T) {
	m, fid := sharingFixtureProofMode(t, ast.ASTProofEnabled)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(980+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	binary, e := m.MkFamilyApp(fid, 4, nil, vars[:2])
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
	sharingMust(t, e)
	f0, e := m.MkEq(binary, vars[2])
	sharingMust(t, e)
	f1, e := m.MkEq(nary, vars[3])
	sharingMust(t, e)
	goal, e := tactic.NewGoal(m, true, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(f0))
	sharingMust(t, goal.Assert(f1))
	before, e := goal.Proof(1)
	sharingMust(t, e)
	pass, e := NewSharingGoalPass(m, 128)
	sharingMust(t, e)
	_, e = pass.Apply(goal)
	sharingMust(t, e)
	form, e := goal.Form(1)
	sharingMust(t, e)
	pr, e := goal.Proof(1)
	sharingMust(t, e)
	if pr.IsNull() || pr == before || form == f1 {
		t.Fatal("rewritten proof goal was not updated", pr, before, form, f1)
	}
	fact, e := m.GetProofFact(pr)
	sharingMust(t, e)
	if fact != form {
		t.Fatal("proof fact does not match rewritten goal", fact, form)
	}
}

func TestSharingGoalPassProofQuantifierIntroduction(t *testing.T) {
	m, fid := sharingFixtureProofMode(t, ast.ASTProofEnabled)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	x, e := m.MkVar(0, sort)
	sharingMust(t, e)
	y, e := m.MkConst(symbol.Numerical(990), sort)
	sharingMust(t, e)
	z, e := m.MkConst(symbol.Numerical(991), sort)
	sharingMust(t, e)
	for _, h := range []ast.ASTHandle{x, y, z} {
		sharingMust(t, m.IncRef(h))
		h := h
		t.Cleanup(func() { sharingMust(t, m.DecRef(h)) })
	}
	binary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{x, y})
	sharingMust(t, e)
	nary, e := m.MkFamilyApp(fid, 4, nil, []ast.ASTHandle{y, z, x, z})
	sharingMust(t, e)
	f0, e := m.MkEq(binary, z)
	sharingMust(t, e)
	f1, e := m.MkEq(nary, z)
	sharingMust(t, e)
	body, e := m.MkAnd(f0, f1)
	sharingMust(t, e)
	q, e := m.MkQuantifier(ast.ASTQuantifierSpec{Forall: true, Sorts: []ast.ASTHandle{sort}, Names: []symbol.Symbol{symbol.Numerical(992)}, Body: body})
	sharingMust(t, e)
	goal, e := tactic.NewGoal(m, true, true, false)
	sharingMust(t, e)
	t.Cleanup(func() { sharingMust(t, goal.Close()) })
	sharingMust(t, goal.Assert(q))
	pass, e := NewSharingGoalPass(m, 128)
	sharingMust(t, e)
	_, e = pass.Apply(goal)
	sharingMust(t, e)
	form, e := goal.Form(0)
	sharingMust(t, e)
	pr, e := goal.Proof(0)
	sharingMust(t, e)
	if form == q || pr.IsNull() {
		t.Fatal("quantifier proof rewrite was not published", form, pr)
	}
	fact, e := m.GetProofFact(pr)
	sharingMust(t, e)
	if fact != form {
		t.Fatal("quantifier proof fact mismatch", fact, form)
	}
}

func sharingFixture(t *testing.T) (*ast.ASTManager, ast.FamilyID) {
	return sharingFixtureProofMode(t, ast.ASTProofDisabled)
}

func sharingFixtureProofMode(t *testing.T, mode ast.ASTProofMode) (*ast.ASTManager, ast.FamilyID) {
	t.Helper()
	b := arena.NewBudget(1 << 30)
	s, e := symbol.NewManager(symbol.Config{Budget: b})
	sharingMust(t, e)
	q, e := numeral.NewRationalService(numeral.RationalServiceConfig{Integers: numeral.IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	sharingMust(t, e)
	m, e := ast.NewASTManager(ast.ASTManagerConfig{Symbols: s, Numerals: q, Budget: b, ProofMode: mode})
	sharingMust(t, e)
	_, e = m.RegisterArithmeticPlugin()
	sharingMust(t, e)
	fid, e := m.RegisterBitVectorPlugin()
	sharingMust(t, e)
	t.Cleanup(func() {
		sharingMust(t, m.Close())
		sharingMust(t, q.Close())
		sharingMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("owner roots", b.Used())
		}
	})
	return m, fid
}

func TestSharingReducerOrderedPairsReuseAndNumeralPosition(t *testing.T) {
	m, fid := sharingFixture(t)
	sort, e := m.BitVectorSort(8)
	sharingMust(t, e)
	vars := make([]ast.ASTHandle, 4)
	for i := range vars {
		vars[i], e = m.MkConst(symbol.Numerical(uint32(900+i)), sort)
		sharingMust(t, e)
		sharingMust(t, m.IncRef(vars[i]))
	}
	t.Cleanup(func() {
		for i := len(vars) - 1; i >= 0; i-- {
			sharingMust(t, m.DecRef(vars[i]))
		}
	})
	num, e := m.BitVectorNumeralUnsigned(3, 8)
	sharingMust(t, e)
	sharingMust(t, m.IncRef(num))
	t.Cleanup(func() { sharingMust(t, m.DecRef(num)) })
	for _, kind := range []int32{4, 6, 31, 33} {
		binary, e := m.MkFamilyApp(fid, kind, nil, vars[:2])
		sharingMust(t, e)
		sharingMust(t, m.IncRef(binary))
		decl, e := m.GetDecl(binary)
		sharingMust(t, e)
		r, e := NewSharingReducer(m, fid, 128)
		sharingMust(t, e)
		_, changed, e := r.ReduceApp(decl, vars[:2])
		sharingMust(t, e)
		if changed {
			t.Fatal("binary source branch must register and return BR_FAILED", kind)
		}
		// The first pair is nonadjacent. Greedy reuse brings it together before
		// the tree grouping, preserving its source order even when reversed.
		got, changed, e := r.ReduceApp(decl, []ast.ASTHandle{vars[1], vars[2], vars[0], vars[3]})
		sharingMust(t, e)
		if !changed {
			t.Fatal("n-ary BV app was not reduced", kind)
		}
		a, e := m.GetArg(got, 0)
		sharingMust(t, e)
		a, e = m.GetArg(a, 0)
		sharingMust(t, e)
		if a != binary {
			t.Fatal("did not reuse ordered registered pair", kind, a, binary)
		}
		gotWithNum, changed, e := r.ReduceApp(decl, []ast.ASTHandle{num, vars[0], vars[1]})
		sharingMust(t, e)
		if !changed {
			t.Fatal("numeral n-ary reduction", kind)
		}
		first, e := m.GetArg(gotWithNum, 0)
		sharingMust(t, e)
		if first != num {
			t.Fatal("first numeral lost its source position", kind)
		}
		sharingMust(t, r.Close())
		sharingMust(t, m.DecRef(binary))
	}
}
