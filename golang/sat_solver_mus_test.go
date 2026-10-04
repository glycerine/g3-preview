// Source: src/sat/sat_mus.cpp:28–229; src/sat/sat_solver.cpp:2072–2083.
// Licensed under the MIT license. Source author: Nikolaj Bjorner, 2014.
package g3

import "testing"

// The clause excludes precisely the assignment satisfying all three positive
// assumptions. Dropping any assumption permits its variable to be false.
func TestSATSolverMUSActualThreeNecessaryAssumptions(t *testing.T) {
	s, ctx, params := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]satLiteral{1, 3, 5}))
	key := "core.minimize"
	satOK(t, ctx.SetBoolName(&params, &key, true))
	satOK(t, s.UpdateParams(&params))
	got, e := s.Check([]satLiteral{0, 2, 4})
	satOK(t, e)
	if got != lFalse {
		t.Fatal(got)
	}
	n, e := s.conflict.core.size()
	satOK(t, e)
	if n != 3 {
		t.Fatal("irreducible core", n)
	}
	seen := [3]bool{}
	for i := uint32(0); i < n; i++ {
		l, e := s.conflict.core.read(i)
		satOK(t, e)
		if l > 4 || l&1 != 0 {
			t.Fatal(l)
		}
		seen[l>>1] = true
	}
	if seen != [3]bool{true, true, true} {
		t.Fatal(seen)
	}
	cfg, e := s.config.State()
	satOK(t, e)
	if !cfg.CoreMinimize || s.mus.isActive {
		t.Fatal("source flet restoration")
	}
	satFullSolverRuntimeAudit(t, s)
}
