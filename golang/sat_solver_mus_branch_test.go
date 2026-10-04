// Source: src/sat/sat_mus.cpp:60–127,210–229; independent source value fixtures.
// Licensed under the MIT license. Source author: Nikolaj Bjorner, 2014.
package g3

import "testing"

func TestSATSolverMUSRedundantCoreAndPartialWindowActualChecks(t *testing.T) {
	for _, partial := range []bool{false, true} {
		s, ctx, params := satSolverCheckFixture(t)
		for i := 0; i < 4; i++ {
			_, e := s.MkVar(true, true)
			satOK(t, e)
		}
		satOK(t, s.MkClause([]satLiteral{1, 3, 5}))
		result, e := s.Check([]satLiteral{0, 2, 4, 6})
		satOK(t, e)
		if result != lFalse {
			t.Fatal(result)
		}
		// A valid but redundant source core input: original a,b,c plus d remains
		// unsatisfiable. This exercises source core replacement after the actual
		// SAT check without claiming the solver originally included d in its proof.
		satOK(t, s.conflict.core.shrink(0))
		for _, v := range []uint32{0, 2, 4, 6} {
			satOK(t, s.conflict.core.push(v))
		}
		key := "core.minimize_partial"
		satOK(t, ctx.SetBoolName(&params, &key, partial))
		satOK(t, s.UpdateParams(&params))
		oldRestarts := s.gc.restarts
		result, e = s.mus.run()
		satOK(t, e)
		if result != lTrue {
			t.Fatal(result)
		}
		count, e := s.conflict.core.size()
		satOK(t, e)
		if count != 3 {
			t.Fatal("redundant d survived", partial, count)
		}
		if yes, e := satMUSContains(&s.conflict.core, 6); e != nil || yes {
			t.Fatal(yes, e)
		}
		want := ^uint32(0)
		if partial {
			want = oldRestarts + 10
		}
		if s.mus.maxNumRestarts != want {
			t.Fatal("source restart window", s.mus.maxNumRestarts, want)
		}
		n, e := s.mus.model.size()
		satOK(t, e)
		if n != 4 {
			t.Fatal("first source SAT witness not retained", n)
		}
		satFullSolverRuntimeAudit(t, s)
	}
}
func TestSATSolverMUSGetCoreUserScopeSwapPopOrder(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.UserPush())
	satOK(t, s.UserPush())
	first, e := s.conflict.userScopeLiterals.read(0)
	satOK(t, e)
	second, e := s.conflict.userScopeLiterals.read(1)
	satOK(t, e)
	// Pure source get_core input coverage. This is not a claim that positive
	// user guards are an independently valid completed solver unsat proof.
	for _, v := range []uint32{first, 0, second, 2} {
		satOK(t, s.conflict.core.push(v))
	}
	satOK(t, s.mus.getCore())
	for i, want := range []uint32{2, 0} {
		got, e := s.mus.core.read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	for i, want := range []uint32{first, second} {
		got, e := s.mus.mus.read(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satFullSolverRuntimeAudit(t, s)
}
