// Source: src/sat/sat_asymm_branch.cpp:62–119; src/sat/sat_solver.cpp:711–716. Licensed under the MIT license.
package g3

import "testing"

func TestSATIntegrityAsymmActualEnabledWrapperSites(t *testing.T) {
	g, _ := satGCFixture(t, 3)
	l := g.lifecycle()
	p := l.propagation
	ctx := p.config.context
	params, e := ctx.NewValue()
	satOK(t, e)
	t.Cleanup(func() { satOK(t, ctx.Del(&params)) })
	a, e := newSATAsymmBranch(l, &params)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, a.Close()) })
	a.debugInvariant = true
	a.debugPropagate = true
	a.debugMissed = true
	before := p.limit.count()
	satOK(t, a.run(true))
	want := uint64(0)
	if satStorageSourceDebug {
		want = 3
	} // propagate checker, entry checker, exit checker.
	if p.limit.count()-before != want {
		t.Fatal("actual three enabled source checker sites", p.limit.count()-before, want)
	}
	if a.reportActive || a.savedConstructed {
		t.Fatal("completed source locals")
	}
}
