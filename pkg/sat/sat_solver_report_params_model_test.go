// Source: src/sat/sat_solver.cpp:1257-1261,2763-2771; src/util/vector.h.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverReportModelConsumerCopySelfAndGuardedReadPrefix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var source RuntimeUintColumn
	source.Bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, source.Finalize()) })
	for _, v := range [2]util.LBool{util.LTrue, util.LFalse} {
		satOK(t, source.Push(uint32(v)))
	}
	satOK(t, s.setModel(&source))
	m, e := s.AppendModel(nil)
	satOK(t, e)
	if len(m) != 2 || m[0] != util.LTrue || m[1] != util.LFalse || !s.Variables.modelIsCurrent {
		t.Fatal(m, s.Variables.modelIsCurrent)
	}
	satOK(t, source.Write(0, uint32(util.LUndef)))
	m, e = s.AppendModel(nil)
	satOK(t, e)
	if m[0] != util.LTrue {
		t.Fatal("source requires copying, not backing stealing", m)
	}
	satOK(t, s.setModel(&s.model))
	n, e := s.model.Size()
	satOK(t, e)
	if n != 0 || s.Variables.modelIsCurrent {
		t.Fatal("source self reset before append", n)
	}

	// Malformed/source-undefined count publishes an uninitialized second word.
	// The actual guarded read must follow the first completed destination push;
	// no allocation failure or source constructor behavior is invented.
	var bad RuntimeUintColumn
	bad.Bind(s.runtime.memory)
	t.Cleanup(func() { satOK(t, bad.Finalize()) })
	satOK(t, bad.Push(uint32(util.LTrue)))
	satOK(t, WriteRuntimeWord(s.runtime.memory, bad.Raw.Block, 4, 2))
	satOK(t, s.mkModel())
	e = s.setModel(&bad)
	if !errors.Is(e, util.ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	n, x := s.model.Size()
	satOK(t, x)
	if n != 1 || !s.Variables.modelIsCurrent {
		t.Fatal("completed append versus final flag publication", n, s.Variables.modelIsCurrent)
	}
	roots := satFullSolverRuntimeRoots(s)
	roots = append(roots, source.Raw.Block, bad.Raw.Block)
	satScalarAuditRoots(t, s.Variables, roots...)
	satOK(t, bad.Finalize())
	satOK(t, source.Finalize())
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
	var expected util.Random
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
