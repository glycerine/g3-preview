// Source: src/ast/rewriter/bool_rewriter.cpp:586-722,939-997;
// src/ast/rewriter/bool_rewriter.h:128-156,167-186.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package rewriter

import (
	"github.com/glycerine/g3/pkg/ast"
	sattactic "github.com/glycerine/g3/pkg/sat/tactic"

	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestBoolRewriterEqualitySourceCasesAndNegation(t *testing.T) {
	m := astTestManager(t)
	rw := boolRewriter{manager: m}
	p := goalTestAtom(t, m, "eq-p")
	q := goalTestAtom(t, m, "eq-q")
	np := goalTestApp(t, m, ast.ASTOpNot, p)
	nq := goalTestApp(t, m, ast.ASTOpNot, q)
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	eq := goalTestApp(t, m, ast.ASTOpIff, p, q)
	unfold := eq
	left := goalTestApp(t, m, ast.ASTOpIff, p, q)
	right := goalTestApp(t, m, ast.ASTOpIff, p, nq)
	left2 := goalTestApp(t, m, ast.ASTOpIff, q, p)
	right2 := goalTestApp(t, m, ast.ASTOpIff, nq, p)
	out := astTestPin(t, m, q)
	for _, tc := range []struct {
		a, b, want ast.ASTHandle
		status     brStatus
	}{
		{p, p, truth, brDone}, {truth, falsity, falsity, brDone},
		{truth, p, p, brDone}, {p, truth, p, brDone}, {falsity, p, np, brDone}, {p, falsity, np, brDone},
		{p, np, falsity, brDone}, {np, p, falsity, brDone}, {np, nq, unfold, brDone},
		{left, right, falsity, brDone}, {right, left, falsity, brDone},
		{left2, right2, falsity, brDone}, {right2, left2, falsity, brDone},
		{p, q, q, brFailed},
	} {
		satOK(t, m.SetRef(out, q))
		status, e := rw.mkEqCore(tc.a, tc.b, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != tc.status || got != tc.want {
			t.Fatal("source equality", tc, status, got)
		}
	}
	negEq := goalTestApp(t, m, ast.ASTOpIff, np, q)
	for _, tc := range []struct {
		arg, want ast.ASTHandle
		status    brStatus
	}{
		{truth, falsity, brDone}, {falsity, truth, brDone}, {np, p, brDone}, {eq, negEq, brDone}, {p, q, brFailed},
	} {
		satOK(t, m.SetRef(out, q))
		status, e := rw.mkNotCore(tc.arg, out)
		satOK(t, e)
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != tc.status || got != tc.want {
			t.Fatal("source negation", tc, status, got)
		}
	}
	satOK(t, rw.mkEq(p, q, out))
	got, e := m.RefGet(out)
	satOK(t, e)
	if got != eq {
		t.Fatal("raw equality fallback", got, eq)
	}
	satOK(t, rw.mkNot(p, out))
	got, e = m.RefGet(out)
	satOK(t, e)
	if got != np {
		t.Fatal("raw negation fallback")
	}
}

func TestBoolRewriterITEValueEqualityStatusesAndSourceStructure(t *testing.T) {
	m := astTestManager(t)
	c := goalTestAtom(t, m, "value-c")
	d := goalTestAtom(t, m, "value-d")
	q := goalTestAtom(t, m, "value-q")
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	nc := goalTestApp(t, m, ast.ASTOpNot, c)
	for _, extra := range []bool{false, true} {
		rw := boolRewriter{manager: m, iteExtraRules: extra}
		out := astTestPin(t, m, q)
		for _, tc := range []struct{ t, e, val, want ast.ASTHandle }{
			{truth, falsity, truth, c}, {truth, falsity, falsity, nc},
			{truth, truth, truth, truth}, {falsity, falsity, truth, falsity},
			{falsity, truth, truth, nc}, {falsity, truth, falsity, c},
		} {
			ite := goalTestApp(t, m, ast.ASTOpIte, c, tc.t, tc.e)
			for _, swap := range []bool{false, true} {
				a, b := ite, tc.val
				if swap {
					a, b = b, a
				}
				status, e := rw.mkEqCore(a, b, out)
				satOK(t, e)
				got, e := m.RefGet(out)
				satOK(t, e)
				if status != brDone || got != tc.want {
					t.Fatal("source value leaves", extra, swap, status, got, tc.want)
				}
			}
		}
		partial := goalTestApp(t, m, ast.ASTOpIte, c, truth, q)
		status, e := rw.mkEqCore(partial, truth, out)
		satOK(t, e)
		if extra {
			want := goalTestApp(t, m, ast.ASTOpOr, c, goalTestApp(t, m, ast.ASTOpIff, truth, q))
			got, e := m.RefGet(out)
			satOK(t, e)
			if status != brRewrite2 || got != want {
				t.Fatal("source partial-leaf expansion", status, got, want)
			}
		} else {
			if status != brDone {
				t.Fatal("source Boolean true equality after failed value shortcut", status)
			}
			got, e := m.RefGet(out)
			satOK(t, e)
			if got != partial {
				t.Fatal("source true equality", got, partial)
			}
		}
		nested := goalTestApp(t, m, ast.ASTOpIte, d, truth, falsity)
		tree := goalTestApp(t, m, ast.ASTOpIte, c, nested, q)
		status, e = rw.mkEqCore(tree, truth, out)
		satOK(t, e)
		want := goalTestApp(t, m, ast.ASTOpIte, c, d, goalTestApp(t, m, ast.ASTOpIff, q, truth))
		got, e := m.RefGet(out)
		satOK(t, e)
		if status != brRewrite2 || got != want {
			t.Fatal("source nested value expansion", status, got, want)
		}
	}
}

func TestBoolRewriterDerivedConnectivesIndependentTruthTables(t *testing.T) {
	for _, op := range []string{"xor", "implies", "nand", "nor"} {
		m := astTestManager(t)
		rw := boolRewriter{manager: m, flat: true}
		p := goalTestAtom(t, m, "derived-p")
		q := goalTestAtom(t, m, "derived-q")
		out := astTestPin(t, m, ast.ASTHandle{})
		var e error
		switch op {
		case "xor":
			e = rw.mkXor(p, q, out)
		case "implies":
			e = rw.mkImplies(p, q, out)
		case "nand":
			e = rw.mkNand([]ast.ASTHandle{p, q}, out)
		case "nor":
			e = rw.mkNor([]ast.ASTHandle{p, q}, out)
		}
		satOK(t, e)
		formula, e := m.RefGet(out)
		satOK(t, e)
		for bits := 0; bits < 4; bits++ {
			a, b := bits&1 != 0, bits&2 != 0
			want := false
			switch op {
			case "xor":
				want = a != b
			case "implies":
				want = !a || b
			case "nand":
				want = !(a && b)
			case "nor":
				want = !(a || b)
			}
			s, _, _ := satSolverCheckFixture(t)
			tr, e := sattactic.NewGoalTranslator(m, s)
			satOK(t, e)
			forms := []ast.ASTHandle{formula}
			for i, h := range []ast.ASTHandle{p, q} {
				if bits&(1<<i) == 0 {
					h, e = m.MkNot(h)
					satOK(t, e)
				}
				forms = append(forms, h)
			}
			satOK(t, tr.Compile(forms))
			r, e := s.Check(nil)
			satOK(t, e)
			if (r == util.LTrue) != want || r == util.LUndef {
				t.Fatal("independent connective", op, bits, r, want)
			}
			satOK(t, tr.Close())
			satOK(t, s.Close())
		}
	}
}

func TestBoolRewriterITEEqualityIndependentPartialBranchTruthTables(t *testing.T) {
	for _, extra := range []bool{false, true} {
		for _, left := range []bool{false, true} {
			for _, leaf := range []bool{false, true} {
				for _, value := range []bool{false, true} {
					m := astTestManager(t)
					rw := boolRewriter{manager: m, iteExtraRules: extra}
					c := goalTestAtom(t, m, "partial-c")
					q := goalTestAtom(t, m, "partial-q")
					v, e := m.MkBoolVal(value)
					satOK(t, e)
					lit, e := m.MkBoolVal(leaf)
					satOK(t, e)
					thenArg, elseArg := q, lit
					if left {
						thenArg, elseArg = lit, q
					}
					ite := goalTestApp(t, m, ast.ASTOpIte, c, thenArg, elseArg)
					original := goalTestApp(t, m, ast.ASTOpIff, ite, v)
					out := astTestPin(t, m, original)
					status, e := rw.mkEqCore(ite, v, out)
					satOK(t, e)
					if status != brDone && status != brRewrite2 {
						t.Fatal("source partial status", status)
					}
					result, e := m.RefGet(out)
					satOK(t, e)
					for bits := 0; bits < 4; bits++ {
						condition, other := bits&1 != 0, bits&2 != 0
						then, els := other, leaf
						if left {
							then, els = leaf, other
						}
						actual := els
						if condition {
							actual = then
						}
						want := actual == value
						for _, formula := range []ast.ASTHandle{original, result} {
							s, _, _ := satSolverCheckFixture(t)
							tr, e := sattactic.NewGoalTranslator(m, s)
							satOK(t, e)
							forms := []ast.ASTHandle{formula}
							for i, h := range []ast.ASTHandle{c, q} {
								if bits&(1<<i) == 0 {
									h, e = m.MkNot(h)
									satOK(t, e)
								}
								forms = append(forms, h)
							}
							satOK(t, tr.Compile(forms))
							r, e := s.Check(nil)
							satOK(t, e)
							if (r == util.LTrue) != want || r == util.LUndef {
								t.Fatal("independent ITE equality", extra, left, leaf, value, bits, r, want)
							}
							satOK(t, tr.Close())
							satOK(t, s.Close())
						}
					}
				}
			}
		}
	}
}
