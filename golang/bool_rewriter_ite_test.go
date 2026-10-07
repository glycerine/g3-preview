// Source: src/ast/rewriter/bool_rewriter.cpp:724-938;
// src/ast/rewriter/bool_rewriter.h:159-163.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import "testing"

func TestBoolRewriterITEExactBranchesAndIndependentTruthTables(t *testing.T) {
	m := astTestManager(t)
	c := goalTestAtom(t, m, "ite-c")
	d := goalTestAtom(t, m, "ite-d")
	k := goalTestAtom(t, m, "ite-k")
	p := goalTestAtom(t, m, "ite-p")
	q := goalTestAtom(t, m, "ite-q")
	vars := []ASTHandle{c, d, k, p, q}
	mk := func(op int32, args ...ASTHandle) ASTHandle { return goalTestApp(t, m, op, args...) }
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	nc, nd, nk := mk(ASTOpNot, c), mk(ASTOpNot, d), mk(ASTOpNot, k)
	nq := mk(ASTOpNot, q)
	choose := func(c, a, b bool) bool {
		if c {
			return a
		}
		return b
	}
	identity := func(v []bool) bool { return choose(v[0], v[3], v[4]) }
	type testcase struct {
		name                       string
		condition, then, els, want ASTHandle
		status                     brStatus
		extra                      bool
		eval                       func([]bool) bool
	}
	cases := []testcase{
		{"unchanged", c, p, q, truth, brFailed, false, identity},
		{"negated-condition", nc, p, q, mk(ASTOpIte, c, q, p), brDone, false, func(v []bool) bool { return choose(!v[0], v[3], v[4]) }},
		{"same-then-condition", c, mk(ASTOpIte, c, p, q), q, mk(ASTOpIte, c, p, q), brDone, false, identity},
		{"same-else-condition", c, p, mk(ASTOpIte, c, q, p), p, brDone, false, func(v []bool) bool { return v[3] }},
		{"true-condition", truth, p, q, p, brDone, false, func(v []bool) bool { return v[3] }},
		{"false-condition", falsity, p, q, q, brDone, false, func(v []bool) bool { return v[4] }},
		{"equal-branches", c, p, p, p, brDone, false, func(v []bool) bool { return v[3] }},
		{"true-false", c, truth, falsity, c, brDone, false, func(v []bool) bool { return v[0] }},
		{"false-true", c, falsity, truth, nc, brDone, false, func(v []bool) bool { return !v[0] }},
		{"true-other", c, truth, q, mk(ASTOpOr, c, q), brDone, false, func(v []bool) bool { return v[0] || v[4] }},
		{"false-other", c, falsity, q, mk(ASTOpAnd, nc, q), brDone, false, func(v []bool) bool { return !v[0] && v[4] }},
		{"other-true", c, q, truth, mk(ASTOpOr, nc, q), brDone, false, func(v []bool) bool { return !v[0] || v[4] }},
		{"other-false", c, q, falsity, mk(ASTOpAnd, c, q), brDone, false, func(v []bool) bool { return v[0] && v[4] }},
		{"else-is-condition", c, q, c, mk(ASTOpAnd, c, q), brDone, false, func(v []bool) bool { return v[0] && v[4] }},
		{"then-is-condition", c, c, q, mk(ASTOpOr, c, q), brDone, false, func(v []bool) bool { return v[0] || v[4] }},
		{"then-negated", c, nq, q, mk(ASTOpIff, c, nq), brDone, false, func(v []bool) bool { return v[0] != v[4] }},
		{"else-negated", c, q, nq, mk(ASTOpIff, c, q), brDone, false, func(v []bool) bool { return v[0] == v[4] }},
		{"extra-then-first", c, mk(ASTOpIte, d, p, q), p, mk(ASTOpIte, mk(ASTOpAnd, c, nd), q, p), brRewrite1, true, func(v []bool) bool { return choose(v[0], choose(v[1], v[3], v[4]), v[3]) }},
		{"extra-then-second", c, mk(ASTOpIte, d, p, q), q, mk(ASTOpIte, mk(ASTOpAnd, c, d), p, q), brRewrite1, true, func(v []bool) bool { return choose(v[0], choose(v[1], v[3], v[4]), v[4]) }},
		{"extra-both-same", c, mk(ASTOpIte, d, p, q), mk(ASTOpIte, k, p, q), mk(ASTOpIte, mk(ASTOpOr, mk(ASTOpAnd, c, d), mk(ASTOpAnd, nc, k)), p, q), brRewrite1, true, func(v []bool) bool { return choose(v[0], choose(v[1], v[3], v[4]), choose(v[2], v[3], v[4])) }},
		{"extra-both-swapped", c, mk(ASTOpIte, d, p, q), mk(ASTOpIte, k, q, p), mk(ASTOpIte, mk(ASTOpOr, mk(ASTOpAnd, c, d), mk(ASTOpAnd, nc, nk)), p, q), brRewrite1, true, func(v []bool) bool { return choose(v[0], choose(v[1], v[3], v[4]), choose(v[2], v[4], v[3])) }},
		{"extra-else-first", c, p, mk(ASTOpIte, d, p, q), mk(ASTOpIte, mk(ASTOpOr, c, d), p, q), brRewrite1, true, func(v []bool) bool { return choose(v[0], v[3], choose(v[1], v[3], v[4])) }},
		{"extra-else-second", c, p, mk(ASTOpIte, d, q, p), mk(ASTOpIte, mk(ASTOpOr, c, nd), p, q), brRewrite1, true, func(v []bool) bool { return choose(v[0], v[3], choose(v[1], v[4], v[3])) }},
	}
	for _, extra := range []bool{false, true} {
		rw := boolRewriter{manager: m, flat: true, iteExtraRules: extra}
		for _, tc := range cases {
			t.Run(tc.name+map[bool]string{false: "-default", true: "-extra"}[extra], func(t *testing.T) {
				original := mk(ASTOpIte, tc.condition, tc.then, tc.els)
				out := astTestPin(t, m, truth)
				status, e := rw.mkITECore(tc.condition, tc.then, tc.els, out)
				satOK(t, e)
				wantStatus, want := tc.status, tc.want
				if tc.extra && !extra {
					wantStatus, want = brFailed, truth
				}
				got, e := m.RefGet(out)
				satOK(t, e)
				if status != wantStatus || got != want {
					t.Fatal("source ITE status/shape", status, wantStatus, got, want)
				}
				satOK(t, rw.mkITE(tc.condition, tc.then, tc.els, out))
				rewritten, e := m.RefGet(out)
				satOK(t, e)
				for bits := 0; bits < 32; bits++ {
					values := []bool{bits&1 != 0, bits&2 != 0, bits&4 != 0, bits&8 != 0, bits&16 != 0}
					want := tc.eval(values)
					for _, formula := range []ASTHandle{original, rewritten} {
						s, _, _ := satSolverCheckFixture(t)
						tr, e := newSATGoalTranslator(m, s)
						satOK(t, e)
						forms := []ASTHandle{formula}
						for i, h := range vars {
							if !values[i] {
								h, e = m.MkNot(h)
								satOK(t, e)
							}
							forms = append(forms, h)
						}
						satOK(t, tr.Compile(forms))
						r, e := s.Check(nil)
						satOK(t, e)
						if (r == lTrue) != want || r == lUndef {
							t.Fatal("independent ITE meaning", bits, r, want)
						}
						satOK(t, tr.Close())
						satOK(t, s.Close())
					}
				}
			})
		}
	}
}

func TestBoolRewriterDistinctCardinalityThresholdAndPairOrder(t *testing.T) {
	m := astTestManager(t)
	rw := boolRewriter{manager: m}
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "distinct-sort"), nil)
	satOK(t, e)
	astTestPin(t, m, s)
	var vars, values []ASTHandle
	for i, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		h, e := m.MkConst(astTestSymbol(t, m, name), s)
		satOK(t, e)
		astTestPin(t, m, h)
		vars = append(vars, h)
		index, e := m.NewParameterInt(int32(i))
		satOK(t, e)
		sortParam, e := m.NewParameterAST(s)
		satOK(t, e)
		v, e := m.MkFamilyApp(3, 0, []ASTParameter{index, sortParam}, nil)
		satOK(t, e)
		astTestPin(t, m, v)
		satOK(t, m.DelParameter(&sortParam))
		satOK(t, m.DelParameter(&index))
		values = append(values, v)
	}
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	out := astTestPin(t, m, truth)
	for _, tc := range []struct {
		args   []ASTHandle
		status brStatus
		want   ASTHandle
	}{
		{nil, brDone, truth}, {vars[:1], brDone, truth},
		{[]ASTHandle{vars[0], vars[0], vars[1]}, brDone, falsity},
		{values[:3], brDone, truth},
		{[]ASTHandle{goalTestAtom(t, m, "bool-a"), goalTestAtom(t, m, "bool-b"), goalTestAtom(t, m, "bool-c")}, brDone, falsity},
		{vars[:3], brFailed, truth},
	} {
		satOK(t, m.SetRef(out, truth))
		status, e := rw.mkDistinctCore(tc.args, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != tc.status || got != tc.want {
			t.Fatal("source distinct shortcut", status, got, tc)
		}
	}
	status, e := rw.mkDistinctCore(vars[:2], out)
	satOK(t, e)
	eq, e := m.MkEq(vars[0], vars[1])
	satOK(t, e)
	neg, e := m.MkNot(eq)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != brRewrite2 || got != neg {
		t.Fatal("two-argument raw distinct", status, got, neg)
	}
	status, e = rw.mkDistinctCore([]ASTHandle{vars[0], vars[0]}, out)
	satOK(t, e)
	eq, e = m.MkEq(vars[0], vars[0])
	satOK(t, e)
	neg, e = m.MkNot(eq)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != brRewrite2 || got != neg {
		t.Fatal("source two-argument shortcut precedes duplicate folding", status, got, neg)
	}
	rw.blastDistinct = true
	rw.blastDistinctThreshold = 3
	satOK(t, m.SetRef(out, truth))
	status, e = rw.mkDistinctCore(vars[:3], out)
	satOK(t, e)
	if status != brFailed {
		t.Fatal("source strict less-than threshold", status)
	}
	rw.blastDistinctThreshold = 9
	status, e = rw.mkDistinctCore(vars, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != brRewrite3 {
		t.Fatal(status)
	}
	n, e := m.GetNumArgs(got)
	satOK(t, e)
	if n != 28 {
		t.Fatal("pair count/spill", n)
	}
	index := uint32(0)
	for i := 0; i < len(vars); i++ {
		for j := i + 1; j < len(vars); j++ {
			child, e := m.GetArg(got, index)
			satOK(t, e)
			index++
			yes, eq, e := m.IsNot(child)
			satOK(t, e)
			if !yes {
				t.Fatal("disequality shape")
			}
			a, e := m.GetArg(eq, 0)
			satOK(t, e)
			b, e := m.GetArg(eq, 1)
			satOK(t, e)
			if a != vars[i] || b != vars[j] {
				t.Fatal("source pair order", i, j, a, b)
			}
		}
	}
	rw.blastDistinct = false
	satOK(t, rw.mkDistinct(vars[:3], out))
	got, e = m.RefGet(out)
	satOK(t, e)
	op, e := m.GetDeclKind(got)
	satOK(t, e)
	if op != ASTOpDistinct {
		t.Fatal("source raw distinct fallback", op)
	}
}

func TestBoolRewriterITENonBooleanBranchesPreserveSourceSortAndShape(t *testing.T) {
	m := astTestManager(t)
	rw := boolRewriter{manager: m, flat: true, iteExtraRules: true}
	c := goalTestAtom(t, m, "sort-c")
	d := goalTestAtom(t, m, "sort-d")
	s, e := m.MkUninterpretedSort(astTestSymbol(t, m, "ite-sort"), nil)
	satOK(t, e)
	astTestPin(t, m, s)
	x, e := m.MkConst(astTestSymbol(t, m, "ite-x"), s)
	satOK(t, e)
	astTestPin(t, m, x)
	y, e := m.MkConst(astTestSymbol(t, m, "ite-y"), s)
	satOK(t, e)
	astTestPin(t, m, y)
	truth, e := m.MkTrue()
	satOK(t, e)
	out := astTestPin(t, m, truth)
	status, e := rw.mkITECore(c, x, y, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != brFailed || got != truth {
		t.Fatal("non-Boolean source fallback", status, got)
	}
	satOK(t, rw.mkITE(c, x, y, out))
	got, e = m.RefGet(out)
	satOK(t, e)
	sort, e := m.GetSort(got)
	satOK(t, e)
	if sort != s {
		t.Fatal("raw ITE changed branch sort")
	}
	nested := goalTestApp(t, m, ASTOpIte, d, x, y)
	status, e = rw.mkITECore(c, nested, y, out)
	satOK(t, e)
	and := goalTestApp(t, m, ASTOpAnd, c, d)
	want := goalTestApp(t, m, ASTOpIte, and, x, y)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != brRewrite1 || got != want {
		t.Fatal("non-Boolean source extra rule", status, got, want)
	}
}
