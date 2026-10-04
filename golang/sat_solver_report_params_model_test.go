// Source: src/sat/sat_solver.cpp:1257-1261,2763-2771; src/util/vector.h.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilhash"
	"testing"
)

func TestSATSolverReportModelConsumerCopySelfAndGuardedReadPrefix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var source satSolverRuntimeUintColumn
	source.bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, source.finalize()) })
	for _, v := range [2]lbool{lTrue, lFalse} {
		satOK(t, source.push(uint32(v)))
	}
	satOK(t, s.setModel(&source))
	m, e := s.AppendModel(nil)
	satOK(t, e)
	if len(m) != 2 || m[0] != lTrue || m[1] != lFalse || !s.variables.modelIsCurrent {
		t.Fatal(m, s.variables.modelIsCurrent)
	}
	satOK(t, source.write(0, uint32(lUndef)))
	m, e = s.AppendModel(nil)
	satOK(t, e)
	if m[0] != lTrue {
		t.Fatal("source requires copying, not backing stealing", m)
	}
	satOK(t, s.setModel(&s.model))
	n, e := s.model.size()
	satOK(t, e)
	if n != 0 || s.variables.modelIsCurrent {
		t.Fatal("source self reset before append", n)
	}

	// Malformed/source-undefined count publishes an uninitialized second word.
	// The actual guarded read must follow the first completed destination push;
	// no allocation failure or source constructor behavior is invented.
	var bad satSolverRuntimeUintColumn
	bad.bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, bad.finalize()) })
	satOK(t, bad.push(uint32(lTrue)))
	satOK(t, satAssignmentWriteWord(s.runtime.memory, bad.raw.block, 4, 2))
	satOK(t, s.mkModel())
	e = s.setModel(&bad)
	if !errors.Is(e, memory.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	n, x := s.model.size()
	satOK(t, x)
	if n != 1 || !s.variables.modelIsCurrent {
		t.Fatal("completed append versus final flag publication", n, s.variables.modelIsCurrent)
	}
	roots := satFullSolverRuntimeRoots(s)
	roots = append(roots, source.raw.block, bad.raw.block)
	satScalarAuditRoots(t, s.variables, roots...)
	satOK(t, bad.finalize())
	satOK(t, source.finalize())
	satFullSolverRuntimeAudit(t, s)

}
func TestSATSolverReportUpdateParamsActualChildrenAndSoleRandom(t *testing.T) {
	s, ctx, p := satSolverCheckFixture(t)
	for _, key := range []string{"resolution", "subsumption", "asymm_branch", "scc"} {
		k := key
		satOK(t, ctx.SetBoolName(&p, &k, false))
	}
	seed := "random_seed"
	satOK(t, ctx.SetUIntName(&p, &seed, 39))
	satOK(t, s.UpdateParams(&p))
	if s.simplifier.resolution || s.simplifier.subsumption || s.asymm.enabled || s.scc.enabled {
		t.Fatal("actual source child parameter dispatch")
	}
	var expected utilhash.Random
	expected.SetSeed(39)
	if got, want := s.decision.rng.Next(), expected.Next(); got != want {
		t.Fatal(got, want)
	}
	copy := *s
	if e := copy.UpdateParams(&p); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if e := s.UpdateParams(&p); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
