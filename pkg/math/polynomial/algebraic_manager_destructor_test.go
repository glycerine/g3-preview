package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// Public TLS operations reach an actual signed-host release refusal. The
// coefficient's allocation extent and source numeral are never rewritten.
func TestAlgebraicManagerStoppedYoungestMemberProtectsOlderOwners(t *testing.T) {
	cfg, _, budget := algebraicManagerFixture(t)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, rt.Close) })
	integerCfg := serviceConfig(budget).Integers
	integerCfg.Memory = rt
	integerCfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: integerCfg})
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, q.CloseChecked) })
	cfg.Numerals = q
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, m.Close) })
	z := q.Integers()
	var n unumeral.Int
	digits := make([]uint32, 40000)
	digits[0], digits[len(digits)-1] = 1, 1
	serviceMust(t, z.SetLimbs(&n, digits))
	serviceMust(t, m.univariate.SetInts(m.addTmp, []unumeral.Int{n}))
	serviceMust(t, z.Del(&n))
	view, e := m.univariate.wholeView(m.addTmp)
	serviceMust(t, e)
	value, e := m.univariate.read(view, 0)
	serviceMust(t, e)
	ref := value.Cell
	extent := z.Cell(ref).Backing.Requested
	coreHeader, ownerHeader := m.AlgebraicRootCore.header, m.header
	older := []UPolynomial{m.evalSignTmp, m.tmp3, m.tmp2, m.isolateTmp1}
	up, pm, binary, intervals := m.univariate, m.polynomial, m.binary, m.intervals
	contextTLSNearMinimum(t, rt)
	before := rt.Stats()
	e = m.Close()
	after := rt.Stats()
	if !errors.Is(e, arena.ErrBounds) || after.LiveBlocks >= before.LiveBlocks || after.PhysicalBytes >= before.PhysicalBytes || m.privateFinalized || m.closed {
		t.Fatal("real youngest source release did not stop", e, before, after)
	}
	if m.AlgebraicRootCore.header != coreHeader || m.header != ownerHeader || m.univariate != up || m.polynomial != pm || m.binary != binary || m.intervals != intervals {
		t.Fatal("youngest stop retired older owner headers")
	}
	if z.Cell(ref).Backing.Requested != extent {
		t.Fatal("allocation extent was repaired or replaced")
	}
	for _, p := range older {
		if _, e := up.Size(p); e != nil {
			t.Fatal("older polynomial retired before youngest", e)
		}
	}
	serviceMust(t, binary.Open())
	if _, e := intervals.Bounds(); e != nil {
		t.Fatal("older interval closed", e)
	}
	if e := pm.ready(); e != nil {
		t.Fatal("older polynomial manager closed", e)
	}
	contextTLSFinish(t, rt, m.Close)
	if !m.closed || m.header != 0 {
		t.Fatal("owning continuation retained manager")
	}
	serviceMust(t, q.CheckOwnerReady())
	serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
}
