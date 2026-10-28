package numeral

// Source: src/util/mpq.h:140-145,324-332,456-482.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalArithmeticEntryMulNUMBeforeDEN(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for alias := 0; alias < 3; alias++ {
			m, _ := rationalManager(t, synch)
			a := Rational{Num: NewInt(2), Den: NewInt(1)}
			b := Rational{Num: NewInt(3), Den: NewInt(1)}
			out := NewRational()
			dst := &out
			if alias == 1 {
				dst = &a
			}
			if alias == 2 {
				dst = &b
			}
			var retired Int
			serviceMust(t, m.Z.SetLimbs(&retired, []uint32{1, 0, 1}))
			stale := retired.Cell
			serviceMust(t, m.Z.Del(&retired))
			dst.Den.Cell = stale
			e := m.Mul(&a, &b, dst)
			got := dst.Num
			if dst.Den.Cell != stale {
				t.Fatal("untouched reached stale DEN", dst.Den, e)
			}
			// The injected dead descriptor has no owned storage; name all other
			// roots before restoring that explicit diagnostic token.
			roots := []*Int{&a.Num, &b.Num, &out.Num}
			for _, den := range []*Int{&a.Den, &b.Den, &out.Den} {
				if den != &dst.Den {
					roots = append(roots, den)
				}
			}
			qopsAuditLiveOwnership(t, m, roots)
			dst.Den.Cell = 0
			serviceMust(t, m.Del(&a))
			serviceMust(t, m.Del(&b))
			serviceMust(t, m.Del(&out))
			if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(6) {
				t.Fatal("source NUM multiplication precedes DEN access", synch, alias, got, e)
			}
		}
	}
}

func TestRationalArithmeticEntryDivNUMBeforeDEN(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a := Rational{Num: NewInt(2), Den: NewInt(1)}
		b := Rational{Num: NewInt(3), Den: NewInt(5)}
		out := NewRational()
		a.Den.Cell = m.Z.FailedRef
		e := m.Div(&a, &b, &out)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
		got := out.Num
		a.Den.Cell = 0
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&out))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != NewInt(10) {
			t.Fatal("source crossproduct NUM precedes a.DEN", synch, got, e)
		}
	}
}

func TestRationalArithmeticEntryDivAliasRetainsRawLocal(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, b := NewRational(), NewRational()
		serviceMust(t, m.Z.SetLimbs(&a.Num, []uint32{17, 3, 1}))
		b.Num = NewInt(3)
		a.Den.Cell = m.Z.FailedRef
		before := m.Z.Cells.Stats().Live
		e := m.Div(&a, &b, &b)
		got := m.Z.Cells.Stats().Live
		state, read := m.RawArithmeticState()
		serviceMust(t, read)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		if e2 := m.SourceFinalize(); !errors.Is(e2, ErrRationalLexicalPending) {
			t.Fatal("source finalize cannot discard raw tmp", e2)
		}
		if e2 := m.Div(&a, &b, &b); !errors.Is(e2, ErrRationalRawArithmeticPending) {
			t.Fatal("alias slot reuse must retain original local", e2)
		}
		a.Den.Cell = 0
		// On the old implementation no raw temporary is retained; cleanup is
		// still performed before reporting the behavioral mismatch.
		serviceMust(t, m.CleanupRawArithmetic())
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != before+1 || !state.Division || state.DivisionEnd || !errors.Is(state.DivisionCause, e) {
			t.Fatal("raw tmp has no source exception destructor", synch, got, before, state, e)
		}
	}
}

func TestRationalArithmeticEntryDivEndGuardAndOwningRetry(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	r := &m.RawArithmetic
	serviceMust(t, m.Z.SetLimbs(&r.Div, []uint32{17, 3, 1}))
	r.DivActive = true
	ref := r.Div.Cell
	original := m.Z.Cell(ref)
	rawArithmeticCorrupt(t, m.Z, &r.Div)
	e := m.rawDivEnd()
	if !errors.Is(e, arena.ErrBounds) || !r.DivActive || !r.divEnd || r.divCursor != 0 || r.Div.Cell != ref {
		t.Fatal("source end reached real physical refusal", r, e)
	}
	qopsAuditLiveOwnership(t, m, nil)
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal(e)
	}
	e = m.CleanupRawArithmetic()
	if !errors.Is(e, arena.ErrBounds) || !r.DivActive || r.divCursor != 0 {
		t.Fatal("owning refusal retains actual root", r, e)
	}
	qopsAuditLiveOwnership(t, m, nil)
	// Restore only the deliberately malformed requested-size fixture, never
	// source arithmetic fields or counters, then retry actual owning cleanup.
	c := m.Z.Cell(ref)
	c.Backing.Requested = original.Backing.Requested
	m.Z.Store(ref, c)
	serviceMust(t, m.CleanupRawArithmetic())
	if r.Pending() {
		t.Fatal("owning continuation retained local", r)
	}
	qopsAuditLiveOwnership(t, m, nil)
}
