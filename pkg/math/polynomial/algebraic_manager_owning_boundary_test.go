package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Close is the explicit Go owning abort/release path. It cannot stand in for
// source ~imp member destruction: the polynomial member has distinct source
// cursors and failure semantics. Keep that distinction when adding the source
// algebraic-manager destructor, rather than relabeling owning cleanup.
func TestAlgebraicManagerOwningCloseDoesNotAdvancePolynomialSource(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	pm, up := m.polynomial, m.univariate
	core := pm.core
	if pm.sourceStarted || pm.sourceFinalized || pm.bodyStage != 0 || core.sourceStage != 0 || core.sourceDecDone != ([2]bool{}) {
		t.Fatal("new owner has source destruction history")
	}
	serviceMust(t, m.Close())
	if !m.closed || !pm.closed || !up.closed {
		t.Fatal("owning release did not retire actual member owners")
	}
	if pm.sourceStarted || pm.sourceFinalized || pm.bodyStage != 0 || core.sourceStage != 0 || core.sourceDecDone != ([2]bool{}) {
		t.Fatal("owning close was relabeled as source destruction")
	}
	serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
}

func TestAlgebraicManagerPrivateVectorStoppedDropOwningCleanup(t *testing.T) {
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
	var number unumeral.Int
	digits := make([]uint32, 40000)
	digits[0], digits[len(digits)-1] = 1, 1
	serviceMust(t, z.SetLimbs(&number, digits))
	serviceMust(t, m.uppers.Resize(1))
	slot, e := numericBinaryRationalManager_binarySlot(m.binary, &m.uppers.state, 0)
	serviceMust(t, e)
	serviceMust(t, m.binary.SetInteger(slot, &number))
	serviceMust(t, z.Del(&number))
	upper, lower, roots := m.uppers, m.lowers, m.roots
	pm, up, binary, intervals := m.polynomial, m.univariate, m.binary, m.intervals
	contextTLSNearMinimum(t, rt)
	before := rt.Stats()
	body := m.Close()
	after := rt.Stats()
	if !errors.Is(body, arena.ErrBounds) || !upper.state.dropPending || upper.state.dropCause != body || upper.closed || m.closed || m.privateFinalized || after.LiveBlocks >= before.LiveBlocks {
		t.Fatal("actual private upper-vector source drop did not stop", body, before, after)
	}
	if m.uppers != upper || m.lowers != lower || m.roots != roots || m.polynomial != pm || m.univariate != up || m.binary != binary || m.intervals != intervals {
		t.Fatal("stopped youngest vector retired older members")
	}
	serviceMust(t, pm.ready())
	serviceMust(t, up.check())
	// Explicit owning cleanup drains each consumed physical block; it must not
	// replay Close against the vector's unchanged sticky source-drop error.
	contextTLSFinish(t, rt, m.Close)
	if !m.closed || !upper.closed || m.header != 0 {
		t.Fatal("actual private vector root was not retired")
	}
	serviceMust(t, q.CheckOwnerReady())
}
