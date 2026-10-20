package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicManagerSourceDestructorSuccessAndUsedOwner(t *testing.T) {
	for _, used := range []bool{false, true} {
		t.Run(fmt.Sprint(used), func(t *testing.T) {
			cfg, _, _ := algebraicManagerFixture(t)
			m, e := NewAlgebraicManager(cfg)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, m.Close()) })
			if used {
				n := unumeral.NewInt(2)
				var value, square Algebraic
				serviceMust(t, m.SetInteger(&value, &n))
				serviceMust(t, m.Root(&value, 2, &value))
				serviceMust(t, m.Power(&value, 2, &square))
				if !errors.Is(m.SourceFinalize(), ErrAlgebraicLive) || m.sourceStarted {
					t.Fatal("source destruction admitted caller roots")
				}
				serviceMust(t, m.Del(&square))
				serviceMust(t, m.Del(&value))
			}
			pm, up, fs := m.polynomial, m.univariate, m.factors
			header, coreHeader := m.header, m.AlgebraicRootCore.header
			serviceMust(t, m.SourceFinalize())
			if !m.sourceFinalized || m.sourceStage != 15 || m.closed || m.header != header || m.AlgebraicRootCore.header != coreHeader || !pm.sourceFinalized || !up.sourceFinalized || !fs.sourceFinalized || m.uppers != nil || m.lowers != nil || m.roots != nil {
				t.Fatal("source member composition/boundary", m.sourceStage)
			}
			serviceMust(t, m.SourceFinalize())
			if _, e = m.RationalManager(); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("destructed owner admitted qm", e)
			}
			if e = m.UpdateParams(nil); !errors.Is(e, arena.ErrClosed) {
				t.Fatal("destructed owner admitted params", e)
			}
			stage, index, cause := m.sourceStage, m.sourceIndex, m.sourceCause
			serviceMust(t, m.Close())
			if !m.closed || m.header != 0 || !fs.closed || !pm.closed || !up.closed || m.sourceStage != stage || m.sourceIndex != index || m.sourceCause != cause || !m.sourceFinalized {
				t.Fatal("owning cleanup changed source history")
			}
			serviceMust(t, cfg.Rationals.Manager.CheckOwnerReady())
		})
	}
}

func algebraicSourceTLSFixture(t *testing.T) (*AlgebraicManager, *umemory.Runtime) {
	t.Helper()
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
	return m, rt
}

func TestAlgebraicManagerSourceDestructorEveryPrivateNUMStop(t *testing.T) {
	// Source phases with actual dynamically owning private NUMs. Other phases
	// have separate child destructor coverage; no source state is fabricated.
	for _, stage := range []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13} {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			m, rt := algebraicSourceTLSFixture(t)
			z := m.config.Numerals.Integers()
			var large unumeral.Int
			digits := make([]uint32, 40000)
			digits[0], digits[len(digits)-1] = 1, 1
			serviceMust(t, z.SetLimbs(&large, digits))
			switch stage {
			case 0, 5, 6, 7, 8:
				ps := map[uint8]UPolynomial{0: m.addTmp, 5: m.evalSignTmp, 6: m.tmp3, 7: m.tmp2, 8: m.isolateTmp1}
				serviceMust(t, m.univariate.SetInts(ps[stage], []unumeral.Int{unumeral.NewInt(7), large, unumeral.NewInt(9)}))
			case 1, 2, 3:
				vs := []*BinaryRationalVector{m.uppers, m.lowers, m.roots}
				v := vs[stage-1]
				serviceMust(t, v.Resize(1))
				slot, e := numericBinaryRationalManager_binarySlot(m.binary, &v.state, 0)
				serviceMust(t, e)
				serviceMust(t, m.binary.SetInteger(slot, &large))
			case 4:
				serviceMust(t, m.univariate.SetInts(m.addTmp, []unumeral.Int{large, unumeral.NewInt(1)}))
				serviceMust(t, m.factors.PushCopy(mustUPolyView(t, m.univariate, m.addTmp), 1))
				serviceMust(t, m.univariate.SetInts(m.addTmp, nil))
			case 9:
				serviceMust(t, z.Copy(&m.isRationalTmp, &large))
			case 10:
				serviceMust(t, m.univariate.set(&m.univariate.basic, []unumeral.Int{large}))
			case 11:
				serviceMust(t, m.polynomial.core.som.add(&large, m.polynomial.core.monomials.unit, false))
			case 12:
				serviceMust(t, m.binary.SetInteger(&m.intervals.MulCurr, &large))
			case 13:
				serviceMust(t, z.Copy(&m.binary.AddMulTmp.Num, &large))
			}
			serviceMust(t, z.Del(&large))
			pm, up, intervals, binary := m.polynomial, m.univariate, m.intervals, m.binary
			contextTLSNearMinimum(t, rt)
			blocks := rt.Stats().LiveBlocks
			stop := m.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || m.sourceStage != stage || m.sourceFinalized || m.closed || rt.Stats().LiveBlocks >= blocks {
				t.Fatal("actual ordered source stop", stop, m.sourceStage, stage)
			}
			if (stage == 0 || stage >= 5 && stage <= 8) && m.sourceIndex != 1 {
				t.Fatal("scoped source coefficient cursor", m.sourceIndex)
			}
			if stage < 11 && pm.sourceStarted {
				t.Fatal("older multivariate source owner advanced")
			}
			if stage < 10 && up.sourceFinalizing {
				t.Fatal("older UP source owner advanced")
			}
			if stage < 12 && intervals != m.intervals {
				t.Fatal("older interval owner replaced")
			}
			if stage < 13 && binary != m.binary {
				t.Fatal("older binary owner replaced")
			}
			blocks = rt.Stats().LiveBlocks
			if m.SourceFinalize() != stop || rt.Stats().LiveBlocks != blocks {
				t.Fatal("source replay crossed first stopped member")
			}
			phase, index := m.sourceStage, m.sourceIndex
			contextTLSFinish(t, rt, m.Close)
			if m.sourceCause != stop || m.sourceStage != phase || m.sourceIndex != index || m.sourceFinalized {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
}

func TestAlgebraicManagerSourceDestructorBorrowedAllocator(t *testing.T) {
	cfg, _, _ := algebraicManagerFixture(t)
	allocator, e := umemory.NewSmallAllocator(umemory.SmallAllocatorConfig{Runtime: cfg.Rationals.Manager.Integers().RuntimeOwner(), Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: polynomialSourceDebug}})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, allocator.Close()) })
	cfg.Allocator = allocator
	m, e := NewAlgebraicManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.Close()) })
	serviceMust(t, m.SourceFinalize())
	if m.ownAllocator {
		t.Fatal("borrowed allocator acquired ownership")
	}
	_, e = allocator.NumFreeObjects()
	serviceMust(t, e)
	serviceMust(t, m.Close())
	_, e = allocator.NumFreeObjects()
	serviceMust(t, e)
}

func TestAlgebraicManagerSourceDestructorOwnedPooledAllocatorStop(t *testing.T) {
	if polynomialSourceDebug {
		t.Skip("debug source allocator bypasses chunks")
	}
	m, rt := algebraicSourceTLSFixture(t)
	pm := m.polynomial.core
	// Real caller polynomial allocations fill the actual parent allocator's pool;
	// releasing every caller leaves only the source's reusable private chunks.
	values := make([]Polynomial, 4096)
	for i := range values {
		n := unumeral.NewInt(int32(i) + 2)
		p, e := pm.MkConstantTake(&n)
		serviceMust(t, e)
		serviceMust(t, pm.IncRef(p))
		values[i] = p
	}
	for _, p := range values {
		serviceMust(t, pm.DecRef(p))
	}
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	stop := m.SourceFinalize()
	if !errors.Is(stop, arena.ErrBounds) || m.sourceStage != 14 || m.sourceFinalized || rt.Stats().LiveBlocks >= before {
		t.Fatal("owned pooled allocator did not stop after all source members", stop, m.sourceStage, before, rt.Stats().LiveBlocks)
	}
	after := rt.Stats().LiveBlocks
	if m.SourceFinalize() != stop || rt.Stats().LiveBlocks != after {
		t.Fatal("allocator source failure replayed")
	}
	contextTLSFinish(t, rt, m.Close)
	if m.sourceStage != 14 || m.sourceCause != stop || m.sourceFinalized {
		t.Fatal("owning allocator cleanup changed source history")
	}
}

func TestAlgebraicManagerSourceDestructorRefusesRetainedPrivateReset(t *testing.T) {
	m, rt := algebraicSourceTLSFixture(t)
	z := m.config.Numerals.Integers()
	var large unumeral.Int
	digits := make([]uint32, 40000)
	digits[0], digits[len(digits)-1] = 1, 1
	serviceMust(t, z.SetLimbs(&large, digits))
	serviceMust(t, m.roots.Resize(1))
	slot, e := numericBinaryRationalManager_binarySlot(m.binary, &m.roots.state, 0)
	serviceMust(t, e)
	serviceMust(t, m.binary.SetInteger(slot, &large))
	serviceMust(t, z.Del(&large))
	contextTLSNearMinimum(t, rt)
	stop := m.roots.Reset()
	if !errors.Is(stop, arena.ErrBounds) || !m.roots.state.dropPending {
		t.Fatal("actual retained reset missing", stop)
	}
	before := rt.Stats()
	upper, lower, root := m.uppers, m.lowers, m.roots
	if !errors.Is(m.SourceFinalize(), ErrAlgebraicPending) || m.sourceStarted || m.sourceStage != 0 || m.uppers != upper || m.lowers != lower || m.roots != root || before != rt.Stats() || m.polynomial.sourceStarted || m.univariate.sourceFinalizing {
		t.Fatal("source admission changed actual pending child")
	}
	contextTLSFinish(t, rt, m.Close)
	if m.sourceStarted || m.sourceStage != 0 {
		t.Fatal("owning abort relabeled as source destructor")
	}
}

func TestAlgebraicManagerSourceDestructorRefusesRetainedBinaryDisplay(t *testing.T) {
	m, rt := algebraicSourceTLSFixture(t)
	z := m.config.Numerals.Integers()
	var a unumeral.BinaryRational
	serviceMust(t, z.SetUint64(&a.Num, 1<<63))
	a.KField = 128
	var prime unumeral.Int
	serviceMust(t, z.Copy(&prime, &a.Num))
	serviceMust(t, z.Del(&prime))
	z.Budget().FailAfter(1)
	var dst [32]byte
	_, body := m.binary.FormatDecimalInto(&a, 8, dst[:])
	z.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) || !m.binary.DisplayState().Pending {
		t.Fatal("actual retained display missing", body)
	}
	before := rt.Stats()
	display := m.binary.DisplayState()
	stop := m.SourceFinalize()
	if stop == nil || m.sourceStarted || m.sourceStage != 0 || m.binary.DisplayState() != display || before != rt.Stats() || m.polynomial.sourceStarted || m.univariate.sourceFinalizing {
		t.Fatal("source admission changed retained display", stop)
	}
	serviceMust(t, m.binary.CleanupDisplay())
	serviceMust(t, z.Del(&a.Num))
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.Close())
}
