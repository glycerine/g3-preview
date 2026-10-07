// Source: src/ast/rewriter/bool_rewriter.cpp:268-579.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package g3

import (
	"fmt"
	"testing"
)

func TestBoolRewriterLocalContextExactOrderCostAndIndependentMeaning(t *testing.T) {
	m := astTestManager(t)
	p, q, r := goalTestAtom(t, m, "local-p"), goalTestAtom(t, m, "local-q"), goalTestAtom(t, m, "local-r")
	mk := func(op int32, a ...ASTHandle) ASTHandle { return goalTestApp(t, m, op, a...) }
	np, nq, nr := mk(ASTOpNot, p), mk(ASTOpNot, q), mk(ASTOpNot, r)
	truth, e := m.MkTrue()
	satOK(t, e)
	choose := func(c, a, b bool) bool {
		if c {
			return a
		}
		return b
	}
	cases := []struct {
		name   string
		args   []ASTHandle
		want   ASTHandle
		status brStatus
		eval   func(bool, bool, bool) bool
	}{
		{"unchanged", []ASTHandle{p, q}, truth, brFailed, func(p, q, r bool) bool { return p || q }},
		{"forward-not-or", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpOr, p, q))}, mk(ASTOpOr, p, nq), brDone, func(p, q, r bool) bool { return p || !(p || q) }},
		{"backward-not-or", []ASTHandle{mk(ASTOpNot, mk(ASTOpOr, p, q)), p}, mk(ASTOpOr, nq, p), brDone, func(p, q, r bool) bool { return !(p || q) || p }},
		{"positive-eq", []ASTHandle{p, mk(ASTOpIff, p, q)}, mk(ASTOpOr, p, nq), brDone, func(p, q, r bool) bool { return p || (p == q) }},
		{"negative-eq", []ASTHandle{np, mk(ASTOpIff, p, q)}, mk(ASTOpOr, np, q), brDone, func(p, q, r bool) bool { return !p || (p == q) }},
		{"negated-eq", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpIff, p, q))}, mk(ASTOpOr, p, q), brDone, func(p, q, r bool) bool { return p || (p != q) }},
		{"positive-ite", []ASTHandle{p, mk(ASTOpIte, p, q, r)}, mk(ASTOpOr, p, r), brDone, func(p, q, r bool) bool { return p || choose(p, q, r) }},
		{"negative-ite", []ASTHandle{np, mk(ASTOpIte, p, q, r)}, mk(ASTOpOr, np, q), brDone, func(p, q, r bool) bool { return !p || choose(p, q, r) }},
		{"forward-ite-produces-or", []ASTHandle{p, mk(ASTOpIte, q, np, r)}, mk(ASTOpOr, p, mk(ASTOpOr, q, r)), brDone, func(p, q, r bool) bool { return p || choose(q, !p, r) }},
		{"backward-ite-produces-or", []ASTHandle{mk(ASTOpIte, q, np, r), p}, mk(ASTOpOr, mk(ASTOpOr, q, r), p), brDone, func(p, q, r bool) bool { return choose(q, !p, r) || p }},
		{"negated-ite", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpIte, p, q, r))}, mk(ASTOpOr, p, nr), brDone, func(p, q, r bool) bool { return p || !choose(p, q, r) }},
		{"nested-complement-absorbed", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpOr, np, q))}, p, brDone, func(p, q, r bool) bool { return p || !(!p || q) }},
		{"nested-all-removed-true", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpOr, p, p))}, truth, brDone, func(p, q, r bool) bool { return true }},
		{"nested-negative-literal", []ASTHandle{np, mk(ASTOpNot, mk(ASTOpOr, np, q))}, mk(ASTOpOr, np, nq), brDone, func(p, q, r bool) bool { return !p || !(!p || q) }},
		{"nested-complement-false", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpOr, p, np))}, p, brDone, func(p, q, r bool) bool { return p || !(p || !p) }},
		{"multiple-survivors", []ASTHandle{p, mk(ASTOpNot, mk(ASTOpOr, p, q, r))}, mk(ASTOpOr, p, mk(ASTOpNot, mk(ASTOpOr, q, r))), brDone, func(p, q, r bool) bool { return p || !(p || q || r) }},
	}
	for _, flat := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(tc.name+fmt.Sprint(flat), func(t *testing.T) {
				rw := boolRewriter{manager: m, flat: flat, localContext: true, localContextLimit: ^uint32(0)}
				original := mk(ASTOpOr, tc.args...)
				out := astTestPin(t, m, truth)
				status, e := rw.mkOrCore(tc.args, out)
				satOK(t, e)
				got, e := m.RefGet(out)
				satOK(t, e)
				wantShape := tc.want
				if flat && tc.name == "forward-ite-produces-or" {
					wantShape = mk(ASTOpOr, p, q, r)
				}
				if flat && tc.name == "backward-ite-produces-or" {
					wantShape = mk(ASTOpOr, r, q, p)
				}
				if status != tc.status || got != wantShape {
					t.Fatal("source local-context status/order", status, tc.status, got, wantShape)
				}
				if tc.name == "unchanged" && rw.localContextCost != 20 {
					t.Fatal("two rounds exact source cost", rw.localContextCost)
				}
				if tc.name == "forward-not-or" && rw.localContextCost != 42 {
					t.Fatal("modified forward/backward rounds cost", rw.localContextCost)
				}
				satOK(t, rw.mkOr(tc.args, out))
				rewritten, e := m.RefGet(out)
				satOK(t, e)
				for bits := 0; bits < 8; bits++ {
					want := tc.eval(bits&1 != 0, bits&2 != 0, bits&4 != 0)
					for _, formula := range []ASTHandle{original, rewritten} {
						s, _, _ := satSolverCheckFixture(t)
						tr, e := newSATGoalTranslator(m, s)
						satOK(t, e)
						forms := []ASTHandle{formula}
						for i, h := range []ASTHandle{p, q, r} {
							if bits&(1<<i) == 0 {
								h, e = m.MkNot(h)
								satOK(t, e)
							}
							forms = append(forms, h)
						}
						satOK(t, tr.Compile(forms))
						answer, e := s.Check(nil)
						satOK(t, e)
						if answer == lUndef || (answer == lTrue) != want {
							t.Fatal("independent local-context meaning", bits, answer, want)
						}
						satOK(t, tr.Close())
						satOK(t, s.Close())
					}
				}
			})
		}
	}
}

func TestBoolRewriterLocalContextEntryLimitWrapAndSpill(t *testing.T) {
	m := astTestManager(t)
	p, q := goalTestAtom(t, m, "limit-p"), goalTestAtom(t, m, "limit-q")
	nq := goalTestApp(t, m, ASTOpNot, q)
	nested := goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, p, q))
	truth, e := m.MkTrue()
	satOK(t, e)
	out := astTestPin(t, m, truth)
	rw := boolRewriter{manager: m, flat: true, localContext: true, localContextLimit: 0}
	status, e := rw.mkOrCore([]ASTHandle{p, nested}, out)
	satOK(t, e)
	got, e := m.RefGet(out)
	satOK(t, e)
	if status != brDone || got != goalTestApp(t, m, ASTOpOr, p, nq) || rw.localContextCost != 42 {
		t.Fatal("limit tested only at entry", status, rw.localContextCost)
	}
	satOK(t, m.SetRef(out, truth))
	status, e = rw.mkOrCore([]ASTHandle{p, nested}, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if status != brFailed || got != truth || rw.localContextCost != 42 {
		t.Fatal("cost exceeds entry limit", status, rw.localContextCost)
	}
	rw.localContextCost = ^uint32(0)
	rw.localContextLimit = ^uint32(0)
	status, e = rw.mkOrCore([]ASTHandle{p, q}, out)
	satOK(t, e)
	if status != brFailed || rw.localContextCost != 19 {
		t.Fatal("source unsigned cost wrap", status, rw.localContextCost)
	}
	var args []ASTHandle
	var counts []uint32
	for i := 0; i < 40; i++ {
		h := goalTestAtom(t, m, fmt.Sprintf("local-spill-%d", i))
		args = append(args, h)
		n, e := m.GetRefCount(h)
		satOK(t, e)
		counts = append(counts, n)
	}
	rw.localContextCost = 0
	satOK(t, m.SetRef(out, truth))
	modified, e := rw.localContextSimplify(args, out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	if modified || got != truth || rw.localContextCost != 400 {
		t.Fatal("unchanged owning vector spill", modified, rw.localContextCost)
	}
	// Published vector arguments are borrowed directly by the actual manager.
	for i, h := range args {
		n, e := m.GetRefCount(h)
		satOK(t, e)
		if n != counts[i] {
			t.Fatal("unchanged local vectors leaked references", i, n, counts[i])
		}
	}
	last := goalTestApp(t, m, ASTOpNot, goalTestApp(t, m, ASTOpOr, args[0], q))
	for i, h := range args {
		counts[i], e = m.GetRefCount(h)
		satOK(t, e)
	}
	modified, e = rw.localContextSimplify(append(args, last), out)
	satOK(t, e)
	got, e = m.RefGet(out)
	satOK(t, e)
	n, e := m.GetNumArgs(got)
	satOK(t, e)
	if !modified || n != 41 {
		t.Fatal("modified vector spill publication", modified, n)
	}
	for i, want := range append(args, nq) {
		h, e := m.GetArg(got, uint32(i))
		satOK(t, e)
		if h != want {
			t.Fatal("source spill survivor order", i, h, want)
		}
	}
	satOK(t, m.ResetRef(out))
	for i, h := range args {
		n, e := m.GetRefCount(h)
		satOK(t, e)
		if n != counts[i] {
			t.Fatal("local vectors leaked references", i, n, counts[i])
		}
	}
	neg, e := NewASTFastMark1(m)
	satOK(t, e)
	defer neg.Close()
	pos, e := NewASTFastMark2(m)
	satOK(t, e)
	defer pos.Close()
	for _, h := range args {
		a, e := neg.IsMarked(h)
		satOK(t, e)
		b, e := pos.IsMarked(h)
		satOK(t, e)
		if a || b {
			t.Fatal("local marks survived teardown")
		}
	}
}

func TestBoolRewriterNestedITEUsesRawNonrecursiveBranches(t *testing.T) {
	m := astTestManager(t)
	p, q, r := goalTestAtom(t, m, "nested-p"), goalTestAtom(t, m, "nested-q"), goalTestAtom(t, m, "nested-r")
	mk := func(op int32, a ...ASTHandle) ASTHandle { return goalTestApp(t, m, op, a...) }
	truth, e := m.MkTrue()
	satOK(t, e)
	falsity, e := m.MkFalse()
	satOK(t, e)
	np, nq := mk(ASTOpNot, p), mk(ASTOpNot, q)
	rw := boolRewriter{manager: m, flat: true, localContext: true, localContextLimit: ^uint32(0)}
	out := astTestPin(t, m, truth)
	for _, tc := range []struct{ c, a, b, w ASTHandle }{
		{truth, p, q, p}, {falsity, p, q, q}, {p, q, q, q}, {p, truth, falsity, p}, {p, falsity, truth, np},
		{p, truth, q, mk(ASTOpOr, p, q)}, {p, falsity, q, mk(ASTOpNot, mk(ASTOpOr, p, nq))},
		{p, q, truth, mk(ASTOpOr, np, q)}, {p, q, falsity, mk(ASTOpNot, mk(ASTOpOr, np, nq))},
		{p, q, p, mk(ASTOpNot, mk(ASTOpOr, np, nq))}, {p, p, q, mk(ASTOpOr, p, q)},
		{p, nq, q, mk(ASTOpIff, p, nq)}, {p, q, nq, mk(ASTOpIff, p, q)}, {p, q, r, mk(ASTOpIte, p, q, r)},
	} {
		satOK(t, rw.mkNestedITE(tc.c, tc.a, tc.b, out))
		got, e := m.RefGet(out)
		satOK(t, e)
		if got != tc.w {
			t.Fatal("source raw nested ITE", got, tc.w)
		}
	}
	if rw.localContextCost != 0 {
		t.Fatal("nested ITE recursively invoked local context", rw.localContextCost)
	}
	sort, e := m.MkUninterpretedSort(astTestSymbol(t, m, "nested-ite-sort"), nil)
	satOK(t, e)
	astTestPin(t, m, sort)
	a, e := m.MkConst(astTestSymbol(t, m, "nested-ite-a"), sort)
	satOK(t, e)
	astTestPin(t, m, a)
	b, e := m.MkConst(astTestSymbol(t, m, "nested-ite-b"), sort)
	satOK(t, e)
	astTestPin(t, m, b)
	satOK(t, rw.mkNestedITE(p, a, b, out))
	got, e := m.RefGet(out)
	satOK(t, e)
	want := mk(ASTOpIte, p, a, b)
	if got != want {
		t.Fatal("non-Boolean source nested ITE", got, want)
	}
}
