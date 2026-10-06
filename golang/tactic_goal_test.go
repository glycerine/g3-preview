// Source: src/tactic/goal.cpp:26-30,111-190,240-329,498-631,656-707;
// src/tactic/goal.h:39-143.
// Licensed under the MIT license. Source author: Leonardo de Moura (2011).
package g3

import "testing"

func goalTestAtom(t *testing.T, m *ASTManager, name string) ASTHandle {
	t.Helper()
	h := proofTestBool(t, m, name)
	astTestPin(t, m, h)
	return h
}
func goalTestApp(t *testing.T, m *ASTManager, op int32, args ...ASTHandle) ASTHandle {
	t.Helper()
	h, e := m.MkFamilyApp(0, op, nil, args)
	astTestMust(t, e)
	astTestPin(t, m, h)
	return h
}
func goalTestNew(t *testing.T, m *ASTManager, proofs, cores bool) *tacticGoal {
	t.Helper()
	g, e := newTacticGoal(m, proofs, true, cores)
	astTestMust(t, e)
	t.Cleanup(func() { astTestMust(t, g.Close()) })
	return g
}
func goalTestForms(t *testing.T, g *tacticGoal, want ...ASTHandle) {
	t.Helper()
	n, e := g.Size()
	astTestMust(t, e)
	if int(n) != len(want) {
		t.Fatalf("source formula count %d want %d", n, len(want))
	}
	for i, h := range want {
		got, e := g.Form(uint32(i))
		astTestMust(t, e)
		if got != h {
			t.Fatalf("source formula[%d] %v want %v", i, got, h)
		}
	}
}

func TestTacticGoalAssertionSplittingAndSourceOrder(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	r := goalTestAtom(t, m, "r")
	np := goalTestApp(t, m, ASTOpNot, p)
	input := goalTestApp(t, m, ASTOpAnd, p, goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, q, goalTestApp(t, m, ASTOpNot, r))), p)
	nq := goalTestApp(t, m, ASTOpNot, q)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(input))
	goalTestForms(t, g, p, nq, r, p)
	// A top-level double negation is not rewritten by quick_process's shortcut.
	double := goalTestApp(t, m, ASTOpNot, np)
	astTestMust(t, g.Assert(double))
	goalTestForms(t, g, p, nq, r, p, double)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	astTestMust(t, g.Assert(truth))
	goalTestForms(t, g, p, nq, r, p, double)
}

func TestTacticGoalFalseReplacesAllAndResetAttributes(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.SetDepth(17))
	astTestMust(t, g.SetPrecision(tacticGoalOver))
	f, e := m.MkFalse()
	astTestMust(t, e)
	astTestMust(t, g.Assert(f))
	astTestMust(t, g.Assert(p))
	goalTestForms(t, g, f)
	if !g.inconsistent || !g.IsDecidedUnsat() {
		t.Fatal("source false goal must decide UNSAT")
	}
	astTestMust(t, g.Reset())
	goalTestForms(t, g)
	if g.depth != 17 || g.precision != tacticGoalOver || g.IsDecidedSat() {
		t.Fatal("reset must preserve over-approximation and depth")
	}
	astTestMust(t, g.ResetAll())
	if g.depth != 0 || g.precision != tacticGoalPrecise || !g.IsDecidedSat() {
		t.Fatal("reset_all restores precise empty goal")
	}
}

func TestTacticGoalUpdateAndEliminateTrue(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	r := goalTestAtom(t, m, "r")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(r))
	input := goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, p, goalTestApp(t, m, ASTOpNot, q)))
	astTestMust(t, g.Update(0, input, ASTHandle{}, ASTDependency{}))
	np := goalTestApp(t, m, ASTOpNot, p)
	goalTestForms(t, g, np, q)
	truth, e := m.MkTrue()
	astTestMust(t, e)
	astTestMust(t, g.Update(0, truth, ASTHandle{}, ASTDependency{}))
	goalTestForms(t, g, truth, q)
	astTestMust(t, g.ElimTrue())
	goalTestForms(t, g, q)
}

func TestTacticGoalRedundanciesProofAndDependency(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "positive-first", true: "negative-first"}[reverse], func(t *testing.T) {
			m := astTestManager(t)
			astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
			p := goalTestAtom(t, m, "p")
			np := goalTestApp(t, m, ASTOpNot, p)
			g := goalTestNew(t, m, true, true)
			a, b := p, np
			if reverse {
				a, b = b, a
			}
			da, e := m.DependencyLeaf(a)
			astTestMust(t, e)
			astTestMust(t, m.DependencyInc(da))
			defer func() { astTestMust(t, m.DependencyDec(da)) }()
			db, e := m.DependencyLeaf(b)
			astTestMust(t, e)
			astTestMust(t, m.DependencyInc(db))
			defer func() { astTestMust(t, m.DependencyDec(db)) }()
			pa, e := m.MkProofAsserted(a)
			astTestMust(t, e)
			astTestPin(t, m, pa)
			pb, e := m.MkProofAsserted(b)
			astTestMust(t, e)
			astTestPin(t, m, pb)
			astTestMust(t, g.AssertWithProof(a, pa, da))
			astTestMust(t, g.AssertWithProof(a, pa, da))
			astTestMust(t, g.AssertWithProof(b, pb, db))
			astTestMust(t, g.ElimRedundancies())
			f, e := m.MkFalse()
			astTestMust(t, e)
			goalTestForms(t, g, f)
			pr, e := g.Proof(0)
			astTestMust(t, e)
			proofTestFact(t, m, pr, f)
			dep, e := g.Dependency(0)
			astTestMust(t, e)
			v, e := m.NewExprVector()
			astTestMust(t, e)
			defer func() { astTestMust(t, m.DelExprVector(&v)) }()
			astTestMust(t, m.DependencyLinearize(dep, v))
			size, e := m.ExprVectorSize(v)
			astTestMust(t, e)
			if size != 2 {
				t.Fatal("source contradiction core must join both premises", size)
			}
		})
	}
}

func TestTacticGoalProofAssertionSplitting(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	input := goalTestApp(t, m, ASTOpAnd, p, goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, q, goalTestApp(t, m, ASTOpNot, p))))
	g := goalTestNew(t, m, true, false)
	astTestMust(t, g.Assert(input))
	nq := goalTestApp(t, m, ASTOpNot, q)
	goalTestForms(t, g, p, nq, p)
	for i, h := range []ASTHandle{p, nq, p} {
		pr, e := g.Proof(uint32(i))
		astTestMust(t, e)
		proofTestFact(t, m, pr, h)
	}
}

func TestTacticGoalCopyAndPrecisionAlgebra(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	g := goalTestNew(t, m, false, false)
	target := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(p))
	astTestMust(t, g.SetDepth(3))
	astTestMust(t, target.SetDepth(7))
	astTestMust(t, g.SetPrecision(tacticGoalUnder))
	astTestMust(t, target.SetPrecision(tacticGoalOver))
	astTestMust(t, g.CopyTo(target))
	goalTestForms(t, target, p)
	if target.depth != 7 || target.precision != tacticGoalUnderOver {
		t.Fatal("source copy attributes", target.depth, target.precision)
	}
	astTestMust(t, g.Reset())
	goalTestForms(t, target, p)
	for a := tacticGoalPrecise; a <= tacticGoalUnderOver; a++ {
		for b := tacticGoalPrecise; b <= tacticGoalUnderOver; b++ {
			want := a
			if a == tacticGoalPrecise {
				want = b
			} else if b != tacticGoalPrecise && a != b {
				want = tacticGoalUnderOver
			}
			if got := tacticGoalPrecisionUnion(a, b); got != want {
				t.Fatal(a, b, got, want)
			}
		}
	}
}

func TestTacticGoalDuplicateEliminationAndRetainedReferences(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	g := goalTestNew(t, m, false, false)
	before, e := m.GetRefCount(p)
	astTestMust(t, e)
	for _, h := range []ASTHandle{p, p, q, p} {
		astTestMust(t, g.Assert(h))
	}
	astTestMust(t, g.ElimRedundancies())
	goalTestForms(t, g, p, q)
	after, e := m.GetRefCount(p)
	astTestMust(t, e)
	if after != before+1 {
		t.Fatal("retained goal array must own only surviving formula", before, after)
	}
	astTestMust(t, g.Close())
	after, e = m.GetRefCount(p)
	astTestMust(t, e)
	if after != before {
		t.Fatal("goal close must release retained formula", before, after)
	}
}

func TestTacticGoalUpdateRetainsSavedFirstBeforeLaterFalse(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(q))
	f, e := m.MkFalse()
	astTestMust(t, e)
	input := goalTestApp(t, m, ASTOpAnd, p, f, q)
	before, e := m.GetRefCount(p)
	astTestMust(t, e)
	astTestMust(t, g.Update(0, input, ASTHandle{}, ASTDependency{}))
	goalTestForms(t, g, f)
	after, e := m.GetRefCount(p)
	astTestMust(t, e)
	if after != before {
		t.Fatal("first replacement expr_ref must remain owned through later false", before, after)
	}
}

func TestTacticGoalUpdateLiteralFalseHeadSourceBehavior(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(q))
	f, e := m.MkFalse()
	astTestMust(t, e)
	input := goalTestApp(t, m, ASTOpAnd, f, p)
	astTestMust(t, g.Update(0, input, ASTHandle{}, ASTDependency{}))
	// goal.cpp:297-298 deliberately pushes original f, rather than the rewritten
	// fr, on this branch. Preserve that observable source behavior.
	goalTestForms(t, g, q, p, input)
	if g.inconsistent {
		t.Fatal("rewritten false must not replace original f in the source branch")
	}
}

func TestTacticGoalAssertionsToSATAndValidatedModel(t *testing.T) {
	for bits := 0; bits < 8; bits++ {
		t.Run(string(rune('0'+bits)), func(t *testing.T) {
			m := astTestManager(t)
			p := goalTestAtom(t, m, "p")
			q := goalTestAtom(t, m, "q")
			r := goalTestAtom(t, m, "r")
			formula := goalTestApp(t, m, ASTOpAnd, goalTestApp(t, m, ASTOpOr, p, q), goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpAnd, p, q)), goalTestApp(t, m, ASTOpIte, r, p, q))
			g := goalTestNew(t, m, false, false)
			astTestMust(t, g.Assert(formula))
			for i, h := range []ASTHandle{p, q, r} {
				if bits&(1<<i) == 0 {
					h = goalTestApp(t, m, ASTOpNot, h)
				}
				astTestMust(t, g.Assert(h))
			}
			astTestMust(t, g.ElimRedundancies())
			formulas, e := g.AppendFormulas(nil)
			astTestMust(t, e)
			s, _, _ := satSolverCheckFixture(t)
			tr, e := newSATGoalTranslator(m, s)
			satOK(t, e)
			t.Cleanup(func() { satOK(t, tr.Close()) })
			satOK(t, tr.Compile(formulas))
			got, e := s.Check(nil)
			satOK(t, e)
			pv, qv, rv := bits&1 != 0, bits&2 != 0, bits&4 != 0
			selected := qv
			if rv {
				selected = pv
			}
			want := lFalse
			if (pv || qv) && !(pv && qv) && selected {
				want = lTrue
			}
			if got != want {
				t.Fatal("original goal assertions", got, want)
			}
			if got == lTrue {
				model, e := s.AppendModel(nil)
				satOK(t, e)
				for i, h := range []ASTHandle{p, q, r} {
					v, e := tr.Variable(h)
					satOK(t, e)
					if v == satNullBoolVar || int(v) >= len(model) || model[v] == lUndef || (model[v] == lTrue) != (bits&(1<<i) != 0) {
						t.Fatal("model violates goal assertion", i, v)
					}
				}
			}
		})
	}
}

func TestTacticGoalProofUpdateKeepsDerivedFacts(t *testing.T) {
	m := astTestManager(t)
	astTestMust(t, m.ToggleProofMode(ASTProofEnabled))
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	g := goalTestNew(t, m, true, false)
	astTestMust(t, g.Assert(q))
	input := goalTestApp(t, m, ASTOpAnd, p, goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, q, goalTestApp(t, m, ASTOpNot, p))))
	pr, e := m.MkProofAsserted(input)
	astTestMust(t, e)
	astTestPin(t, m, pr)
	astTestMust(t, g.Update(0, input, pr, ASTDependency{}))
	nq := goalTestApp(t, m, ASTOpNot, q)
	goalTestForms(t, g, p, nq, p)
	for i, h := range []ASTHandle{p, nq, p} {
		pr, e := g.Proof(uint32(i))
		astTestMust(t, e)
		proofTestFact(t, m, pr, h)
	}
}

func TestTacticGoalSourceDepthPrecisionCountsAndReferences(t *testing.T) {
	m := astTestManager(t)
	p := goalTestAtom(t, m, "p")
	q := goalTestAtom(t, m, "q")
	f := goalTestApp(t, m, ASTOpOr, p, q)
	g := goalTestNew(t, m, false, false)
	astTestMust(t, g.Assert(f))
	astTestMust(t, g.Assert(f))
	count, e := g.NumExprs()
	astTestMust(t, e)
	if count != 3 {
		t.Fatal("source count must visit shared ASTs once", count)
	}
	astTestMust(t, g.SetDepth((1<<26)-1))
	astTestMust(t, g.IncDepth())
	if g.depth != 0 {
		t.Fatal("source 26-bit depth wraps", g.depth)
	}
	for precision := tacticGoalPrecise; precision <= tacticGoalUnderOver; precision++ {
		astTestMust(t, g.Reset())
		astTestMust(t, g.SetPrecision(precision))
		if g.IsDecidedSat() != (precision == tacticGoalPrecise || precision == tacticGoalUnder) {
			t.Fatal("SAT preservation", precision)
		}
		f, e := m.MkFalse()
		astTestMust(t, e)
		astTestMust(t, g.Assert(f))
		if g.IsDecidedUnsat() != (precision == tacticGoalPrecise || precision == tacticGoalOver) {
			t.Fatal("UNSAT preservation", precision)
		}
	}
	astTestMust(t, g.IncRef())
	astTestMust(t, g.DecRef())
	if _, e = g.Size(); e == nil {
		t.Fatal("source final reference must retire goal")
	}
}
