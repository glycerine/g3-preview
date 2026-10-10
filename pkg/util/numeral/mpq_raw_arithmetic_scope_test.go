package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// Explicit malformed physical requested-size guard, not source-valid malloc
// exhaustion: actual ILP32 deallocation refuses before physical release.
func rawArithmeticGuardOwner(t *testing.T) (*RationalManager, *arena.Budget, *umemory.Runtime) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	m, e := NewRationalManager(RationalConfig{Synchronized: true, Integers: IntegerConfig{Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	return m, b, r
}
func rawArithmeticCorrupt(t *testing.T, z *IntegerManager, p *Int) {
	t.Helper()
	ref := p.Cell
	original := z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, bad)
	t.Cleanup(func() {
		if ValidNumericRef(z.Cells, ref) {
			c := z.Cell(ref)
			c.Backing.Requested = original.Backing.Requested
			z.Store(ref, c)
		}
	})
}
func TestRationalRawPowerBodyExceptionRetainsLocal(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, out := NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617"))
	serviceMust(t, m.SetString(&out, "18446744073709551619"))
	rawArithmeticCorrupt(t, m.Z, &out.Num)
	before := m.Z.Cells.Stats().Live
	e := m.Power(&a, 0, &out)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if got := m.Z.Cells.Stats().Live; got != before+1 {
		t.Fatalf("raw power has no source exception destructor: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Power || state.PowerEnd || state.Normalization || state.Addition || !errors.Is(state.PowerCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
	if err = m.SourceFinalize(); !errors.Is(err, ErrRationalLexicalPending) {
		t.Fatal(err)
	}
	if err = m.Power(&a, 0, &out); !errors.Is(err, ErrRationalRawArithmeticPending) {
		t.Fatal(err)
	}
	serviceMust(t, m.CleanupRawArithmetic())
	if m.RawArithmetic.Pending() {
		t.Fatal("explicit owning cleanup did not finish")
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
}
func TestRationalRawNormalizeBodyExceptionRetainsLocal(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a := NewRational()
	serviceMust(t, m.Z.SetLimbs(&a.Num, []uint32{17, 0, 1}))
	serviceMust(t, m.Z.Copy(&a.Den, &a.Num))
	rawArithmeticCorrupt(t, m.Z, &a.Num)
	before := m.Z.Cells.Stats().Live
	e := m.normalize(&a)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if got := m.Z.Cells.Stats().Live; got != before+1 {
		t.Fatalf("raw normalize has no source exception destructor: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Normalization || state.NormalizationEnd || !errors.Is(state.NormalizationCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
	if err = m.normalize(&a); !errors.Is(err, ErrRationalRawArithmeticPending) {
		t.Fatal(err)
	}
	serviceMust(t, m.CleanupRawArithmetic())
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den})
}
func TestRationalRawAddBodyExceptionRetainsLocals(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, b, out := NewRational(), NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617/3"))
	serviceMust(t, m.SetString(&b, "1/2"))
	out.Den = Int{Val: 1, Cell: m.Z.FailedRef}
	before := m.Z.Cells.Stats().Live
	e := m.Add(&a, &b, &out)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if got := m.Z.Cells.Stats().Live; got != before+1 {
		t.Fatalf("raw tmp1 must survive third Mul refusal: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Addition || state.AdditionEnd || !errors.Is(state.AdditionCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
	serviceMust(t, m.CleanupRawArithmetic())
	qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
}

func TestRationalRawArithmeticEndGuardAndRetry(t *testing.T) {
	for _, kind := range []uint8{0, 1, 2} {
		m, _, _ := rawArithmeticGuardOwner(t)
		r := &m.RawArithmetic
		var p *Int
		switch kind {
		case 0:
			r.Power = NewRational()
			r.PowerActive = true
			p = &r.Power.Num
		case 1:
			r.AddActive = true
			p = &r.AddOne
		case 2:
			r.NormActive = true
			p = &r.Norm
		}
		serviceMust(t, m.Z.SetLimbs(p, []uint32{17, 0, 1}))
		ref := p.Cell
		original := m.Z.Cell(ref)
		rawArithmeticCorrupt(t, m.Z, p)
		var e error
		switch kind {
		case 0:
			e = m.rawPowerEnd()
		case 1:
			e = m.rawAddEnd()
		case 2:
			e = m.rawNormalizeEnd()
		}
		if !errors.Is(e, arena.ErrBounds) || p.Cell != ref {
			t.Fatal("source end guard discarded actual root", e)
		}
		if kind == 0 && !r.powerEnd || kind == 1 && !r.addEnd || kind == 2 && !r.normEnd {
			t.Fatal("source end label not recorded")
		}
		state, err := m.RawArithmeticState()
		serviceMust(t, err)
		if kind == 0 && (state.PowerGuard || state.PowerCursor != 0) || kind == 1 && (state.AdditionGuard || state.AdditionCursor != 0) || kind == 2 && state.NormalizationGuard {
			t.Fatal("source end deallocation refusal/cursor history", kind, state)
		}
		qopsAuditLiveOwnership(t, m, nil)
		if e = m.CleanupRawArithmetic(); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("owning guard bypassed malformed layout", e)
		}
		qopsAuditLiveOwnership(t, m, nil)
		// Restore only explicitly malformed test allocation metadata. No arithmetic
		// fields or allocator counters are changed, and no valid source failure repaired.
		cell := m.Z.Cell(ref)
		cell.Backing.Requested = original.Backing.Requested
		m.Z.Store(ref, cell)
		serviceMust(t, m.CleanupRawArithmetic())
		qopsAuditLiveOwnership(t, m, nil)
	}
}

func TestRationalRawOwningCloseStopsOwnerEntry(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	r := &m.RawArithmetic
	r.NormActive = true
	serviceMust(t, m.Z.SetLimbs(&r.Norm, []uint32{17, 0, 1}))
	ref := r.Norm.Cell
	original := m.Z.Cell(ref)
	rawArithmeticCorrupt(t, m.Z, &r.Norm)
	if e := m.CloseChecked(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if r.Norm.Cell != ref || !r.NormActive {
		t.Fatal("owning close lost stopped root")
	}
	qopsAuditLiveOwnership(t, m, nil)
	x := NewRational()
	if _, e := m.IsInt(&x); !errors.Is(e, arena.ErrClosed) {
		t.Error("IsInt after owning close", e)
	}
	if e := m.SetInt32(&x, 2); !errors.Is(e, arena.ErrClosed) {
		t.Error("SetInt32 after owning close", e)
	}
	if e := m.Add(&x, &x, &x); !errors.Is(e, arena.ErrClosed) {
		t.Error("Add after owning close", e)
	}
	if e := m.Mul(&x, &x, &x); !errors.Is(e, arena.ErrClosed) {
		t.Error("Mul after owning close", e)
	}
	c := m.Z.Cell(ref)
	c.Backing.Requested = original.Backing.Requested
	m.Z.Store(ref, c)
	serviceMust(t, m.CloseChecked())
}
