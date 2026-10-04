// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"testing"
)

// Explicit malformed physical requested-size guard, not source-valid malloc
// exhaustion: actual ILP32 deallocation refuses before physical release.
func rawArithmeticGuardOwner(t *testing.T) (*RationalManager, *arena.Budget, *memory.Runtime) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	m, e := NewRationalManager(RationalConfig{Synchronized: true, Integers: IntegerConfig{Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 32, Debug: true}, Cells: arena.Config{PageSize: 2, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
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
	ref := p.cell
	original := z.cell(ref)
	bad := original
	bad.backing.requested = uint64(math.MaxUint32) + 1
	z.store(ref, bad)
	t.Cleanup(func() {
		if validNumericRef(z.cells, ref) {
			c := z.cell(ref)
			c.backing.requested = original.backing.requested
			z.store(ref, c)
		}
	})
}
func TestRationalRawPowerBodyExceptionRetainsLocal(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, out := NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617"))
	serviceMust(t, m.SetString(&out, "18446744073709551619"))
	rawArithmeticCorrupt(t, m.z, &out.num)
	before := m.z.cells.Stats().Live
	e := m.Power(&a, 0, &out)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if got := m.z.cells.Stats().Live; got != before+1 {
		t.Fatalf("raw power has no source exception destructor: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Power || state.PowerEnd || state.Normalization || state.Addition || !errors.Is(state.PowerCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den})
	if err = m.SourceFinalize(); !errors.Is(err, ErrRationalLexicalPending) {
		t.Fatal(err)
	}
	if err = m.Power(&a, 0, &out); !errors.Is(err, ErrRationalRawArithmeticPending) {
		t.Fatal(err)
	}
	serviceMust(t, m.CleanupRawArithmetic())
	if m.rawArithmetic.pending() {
		t.Fatal("explicit owning cleanup did not finish")
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den})
}
func TestRationalRawNormalizeBodyExceptionRetainsLocal(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a := NewRational()
	serviceMust(t, m.z.SetLimbs(&a.num, []uint32{17, 0, 1}))
	serviceMust(t, m.z.Copy(&a.den, &a.num))
	rawArithmeticCorrupt(t, m.z, &a.num)
	before := m.z.cells.Stats().Live
	e := m.normalize(&a)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if got := m.z.cells.Stats().Live; got != before+1 {
		t.Fatalf("raw normalize has no source exception destructor: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Normalization || state.NormalizationEnd || !errors.Is(state.NormalizationCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
	if err = m.normalize(&a); !errors.Is(err, ErrRationalRawArithmeticPending) {
		t.Fatal(err)
	}
	serviceMust(t, m.CleanupRawArithmetic())
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
}
func TestRationalRawAddBodyExceptionRetainsLocals(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	a, b, out := NewRational(), NewRational(), NewRational()
	serviceMust(t, m.SetString(&a, "18446744073709551617/3"))
	serviceMust(t, m.SetString(&b, "1/2"))
	out.den = Int{val: 1, cell: m.z.failedRef}
	before := m.z.cells.Stats().Live
	e := m.Add(&a, &b, &out)
	if !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if got := m.z.cells.Stats().Live; got != before+1 {
		t.Fatalf("raw tmp1 must survive third Mul refusal: live=%d expected=%d", got, before+1)
	}
	state, err := m.RawArithmeticState()
	serviceMust(t, err)
	if !state.Addition || state.AdditionEnd || !errors.Is(state.AdditionCause, e) {
		t.Fatal(state)
	}
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
	serviceMust(t, m.CleanupRawArithmetic())
	qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
}

func TestRationalRawArithmeticEndGuardAndRetry(t *testing.T) {
	for _, kind := range []uint8{0, 1, 2} {
		m, _, _ := rawArithmeticGuardOwner(t)
		r := &m.rawArithmetic
		var p *Int
		switch kind {
		case 0:
			r.power = NewRational()
			r.powerActive = true
			p = &r.power.num
		case 1:
			r.addActive = true
			p = &r.addOne
		case 2:
			r.normActive = true
			p = &r.norm
		}
		serviceMust(t, m.z.SetLimbs(p, []uint32{17, 0, 1}))
		ref := p.cell
		original := m.z.cell(ref)
		rawArithmeticCorrupt(t, m.z, p)
		var e error
		switch kind {
		case 0:
			e = m.rawPowerEnd()
		case 1:
			e = m.rawAddEnd()
		case 2:
			e = m.rawNormalizeEnd()
		}
		if !errors.Is(e, arena.ErrBounds) || p.cell != ref {
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
		cell := m.z.cell(ref)
		cell.backing.requested = original.backing.requested
		m.z.store(ref, cell)
		serviceMust(t, m.CleanupRawArithmetic())
		qopsAuditLiveOwnership(t, m, nil)
	}
}

func TestRationalRawOwningCloseStopsOwnerEntry(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	r := &m.rawArithmetic
	r.normActive = true
	serviceMust(t, m.z.SetLimbs(&r.norm, []uint32{17, 0, 1}))
	ref := r.norm.cell
	original := m.z.cell(ref)
	rawArithmeticCorrupt(t, m.z, &r.norm)
	if e := m.CloseChecked(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if r.norm.cell != ref || !r.normActive {
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
	c := m.z.cell(ref)
	c.backing.requested = original.backing.requested
	m.z.store(ref, c)
	serviceMust(t, m.CloseChecked())
}
