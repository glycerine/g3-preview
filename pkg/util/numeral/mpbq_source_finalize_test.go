package numeral

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func binarySourceTLSFixture(t *testing.T) (*BinaryRationalManager, *umemory.Runtime) {
	t.Helper()
	budget := arena.NewBudget(1 << 26)
	t.Cleanup(func() {
		if budget.Used() != 0 {
			t.Errorf("retained owner bytes %d", budget.Used())
		}
	})
	s, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, s.Close()) })
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, rt.Close) })
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	z, e := NewIntegerManager(cfg)
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, z.CloseChecked) })
	m, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: s})
	serviceMust(t, e)
	t.Cleanup(func() { contextTLSFinish(t, rt, m.CloseChecked) })
	return m, rt
}

func TestBinaryManagerSourceDestructorSuccessAndOwningBoundary(t *testing.T) {
	m, _ := binaryManager(t)
	// Populate every actual C++ scratch member using its real integer owner.
	roots := []*Int{&m.AddMulTmp.Num, &m.Tmp, &m.Tmp2, &m.SelectIntTmp1, &m.SelectIntTmp2, &m.SelectSmallTmp, &m.SelectSmallTmp1.Num, &m.SelectSmallTmp2.Num, &m.DivTmp1, &m.DivTmp2, &m.DivTmp3}
	for _, p := range roots {
		serviceMust(t, m.Z.SetInt64(p, 1<<50))
	}
	header := m.HeaderBytes
	serviceMust(t, m.SourceFinalize())
	if !m.sourceFinalized || m.sourceStage != 11 || m.cleanupStage != 0 || m.HeaderBytes != header || m.closed {
		t.Fatal("source destructor crossed owning boundary")
	}
	for _, p := range roots {
		if p.Cell != 0 {
			t.Fatal("source member survived", p)
		}
	}
	if !errors.Is(m.Open(), arena.ErrClosed) {
		t.Fatal("destructed owner admitted use")
	}
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.CloseChecked())
	if !m.closed || m.HeaderBytes != 0 || !m.sourceFinalized || m.sourceStage != 11 {
		t.Fatal("owning close lost source history")
	}
}

func TestBinaryManagerSourceDestructorEveryStoppedMember(t *testing.T) {
	for stage := 0; stage < 11; stage++ {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			m, rt := binarySourceTLSFixture(t)
			roots := []*Int{&m.AddMulTmp.Num, &m.Tmp, &m.Tmp2, &m.SelectIntTmp1, &m.SelectIntTmp2, &m.SelectSmallTmp, &m.SelectSmallTmp1.Num, &m.SelectSmallTmp2.Num, &m.DivTmp1, &m.DivTmp2, &m.DivTmp3}
			digits := make([]uint32, 40000)
			digits[0], digits[len(digits)-1] = 1, 1
			serviceMust(t, m.Z.SetLimbs(roots[stage], digits))
			if stage+1 < len(roots) {
				serviceMust(t, m.Z.SetInt64(roots[stage+1], 1<<50))
			}
			before := make([]Int, len(roots))
			for i, p := range roots {
				before[i] = *p
			}
			contextTLSNearMinimum(t, rt)
			blocks := rt.Stats().LiveBlocks
			stop := m.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || m.sourceStage != uint8(stage) || m.sourceFinalized || m.cleanupStage != 0 || rt.Stats().LiveBlocks >= blocks {
				t.Fatal("actual source stop", stop, m.sourceStage)
			}
			if *roots[stage] != before[stage] {
				t.Fatal("source release rewrote stopped descriptor")
			}
			for i := stage + 1; i < len(roots); i++ {
				if *roots[i] != before[i] {
					t.Fatal("older member changed", i)
				}
			}
			blocks = rt.Stats().LiveBlocks
			if m.SourceFinalize() != stop || rt.Stats().LiveBlocks != blocks {
				t.Fatal("sticky source release replayed")
			}
			contextTLSFinish(t, rt, m.CloseChecked)
			if m.sourceCause != stop || m.sourceStage != uint8(stage) || m.sourceFinalized || !m.closed {
				t.Fatal("owning cleanup changed source history")
			}
		})
	}
}

func TestBinaryIntervalSourceDestructorEveryStoppedMember(t *testing.T) {
	for stage := 0; stage < 3; stage++ {
		t.Run(fmt.Sprint(stage), func(t *testing.T) {
			b, rt := binarySourceTLSFixture(t)
			m, e := NewBinaryIntervalManager(b)
			serviceMust(t, e)
			t.Cleanup(func() { contextTLSFinish(t, rt, m.CloseChecked) })
			roots := []*BinaryRational{&m.MulCurr, &m.MulMax, &m.MulMin}
			digits := make([]uint32, 40000)
			digits[0], digits[len(digits)-1] = 1, 1
			serviceMust(t, b.Z.SetLimbs(&roots[stage].Num, digits))
			if stage+1 < len(roots) {
				serviceMust(t, b.SetInt32(roots[stage+1], 7))
			}
			before := []BinaryRational{m.MulCurr, m.MulMax, m.MulMin}
			contextTLSNearMinimum(t, rt)
			stop := m.SourceFinalize()
			if !errors.Is(stop, arena.ErrBounds) || m.sourceStage != uint8(stage) || m.sourceFinalized || m.cleanupStage != 0 || b.sourceStarted {
				t.Fatal("source member stop/order", stop, m.sourceStage)
			}
			for i := stage; i < len(roots); i++ {
				if *roots[i] != before[i] {
					t.Fatal("stopped or older member changed", i)
				}
			}
			blocks := rt.Stats().LiveBlocks
			if m.SourceFinalize() != stop || rt.Stats().LiveBlocks != blocks {
				t.Fatal("sticky source replay")
			}
			contextTLSFinish(t, rt, m.CloseChecked)
			if m.sourceCause != stop || m.sourceStage != uint8(stage) || b.sourceStarted {
				t.Fatal("owning cleanup changed source or borrowed owner")
			}
			serviceMust(t, b.Open())
		})
	}
}

func TestBinaryIntervalSourceDestructorSuccess(t *testing.T) {
	b, _ := binaryManager(t)
	m, e := NewBinaryIntervalManager(b)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	for _, p := range []*BinaryRational{&m.MulCurr, &m.MulMax, &m.MulMin} {
		serviceMust(t, b.SetInt32(p, 7))
	}
	header := m.HeaderBytes
	serviceMust(t, m.SourceFinalize())
	if !m.sourceFinalized || m.sourceStage != 3 || m.HeaderBytes != header || m.cleanupStage != 0 || m.closed || b.sourceStarted {
		t.Fatal("source boundary")
	}
	serviceMust(t, b.Open())
	serviceMust(t, m.SourceFinalize())
	serviceMust(t, m.CloseChecked())
	if !m.closed || !m.sourceFinalized || m.sourceStage != 3 {
		t.Fatal("owning boundary")
	}
}

func TestBinaryManagerSourceDestructorRefusesActualRetainedDisplay(t *testing.T) {
	z, budget := mpzPublicationDirectOwner(t)
	m, e := binaryDisplayTestOwner(t, z)
	serviceMust(t, e)
	defer m.Close()
	var a BinaryRational
	serviceMust(t, z.SetUint64(&a.Num, 1<<63))
	a.KField = 128
	defer z.Del(&a.Num)
	var prime Int
	serviceMust(t, z.Copy(&prime, &a.Num))
	serviceMust(t, z.Del(&prime))
	budget.FailAfter(1)
	var dst [32]byte
	_, body := m.FormatDecimalInto(&a, 8, dst[:])
	budget.FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) || !m.Display.active {
		t.Fatal("actual retained display missing", body)
	}
	before := m.Display
	if !errors.Is(m.SourceFinalize(), ErrBinaryDisplayPending) || m.sourceStarted || m.Display != before {
		t.Fatal("source destructor acquired retained display")
	}
	serviceMust(t, m.CleanupDisplay())
	serviceMust(t, m.SourceFinalize())
}

func TestBinaryManagersOwningCloseDoesNotAdvanceSource(t *testing.T) {
	b, _ := binaryManager(t)
	m, e := NewBinaryIntervalManager(b)
	serviceMust(t, e)
	serviceMust(t, b.SetInt32(&m.MulCurr, 7))
	serviceMust(t, b.Z.SetInt64(&b.Tmp, 1<<50))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, b.CloseChecked())
	if m.sourceStarted || m.sourceFinalized || m.sourceStage != 0 || m.sourceCause != nil || b.sourceStarted || b.sourceFinalized || b.sourceStage != 0 || b.sourceCause != nil {
		t.Fatal("owning close was relabeled as source destruction")
	}
}
