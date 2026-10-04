// Source: src/util/mpq.h:140-145,324-332,456-482. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func TestRationalArithmeticEntryMulNUMBeforeDEN(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for alias := 0; alias < 3; alias++ {
			m, _ := rationalManager(t, synch)
			a := Rational{num: NewInt(2), den: NewInt(1)}
			b := Rational{num: NewInt(3), den: NewInt(1)}
			out := NewRational()
			dst := &out
			if alias == 1 {
				dst = &a
			}
			if alias == 2 {
				dst = &b
			}
			var retired Int
			serviceMust(t, m.z.SetLimbs(&retired, []uint32{1, 0, 1}))
			stale := retired.cell
			serviceMust(t, m.z.Del(&retired))
			dst.den.cell = stale
			e := m.Mul(&a, &b, dst)
			got := dst.num
			if dst.den.cell != stale {
				t.Fatal("untouched reached stale DEN", dst.den, e)
			}
			// The injected dead descriptor has no owned storage; name all other
			// roots before restoring that explicit diagnostic token.
			roots := []*Int{&a.num, &b.num, &out.num}
			for _, den := range []*Int{&a.den, &b.den, &out.den} {
				if den != &dst.den {
					roots = append(roots, den)
				}
			}
			qopsAuditLiveOwnership(t, m, roots)
			dst.den.cell = 0
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
		a := Rational{num: NewInt(2), den: NewInt(1)}
		b := Rational{num: NewInt(3), den: NewInt(5)}
		out := NewRational()
		a.den.cell = m.z.failedRef
		e := m.Div(&a, &b, &out)
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
		got := out.num
		a.den.cell = 0
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
		serviceMust(t, m.z.SetLimbs(&a.num, []uint32{17, 3, 1}))
		b.num = NewInt(3)
		a.den.cell = m.z.failedRef
		before := m.z.cells.Stats().Live
		e := m.Div(&a, &b, &b)
		got := m.z.cells.Stats().Live
		state, read := m.RawArithmeticState()
		serviceMust(t, read)
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den})
		if e2 := m.SourceFinalize(); !errors.Is(e2, ErrRationalLexicalPending) {
			t.Fatal("source finalize cannot discard raw tmp", e2)
		}
		if e2 := m.Div(&a, &b, &b); !errors.Is(e2, ErrRationalRawArithmeticPending) {
			t.Fatal("alias slot reuse must retain original local", e2)
		}
		a.den.cell = 0
		// On the old implementation no raw temporary is retained; cleanup is
		// still performed before reporting the behavioral mismatch.
		serviceMust(t, m.CleanupRawArithmetic())
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den})
		serviceMust(t, m.Del(&a))
		serviceMust(t, m.Del(&b))
		if !errors.Is(e, arena.ErrInvalidSlot) || got != before+1 || !state.Division || state.DivisionEnd || !errors.Is(state.DivisionCause, e) {
			t.Fatal("raw tmp has no source exception destructor", synch, got, before, state, e)
		}
	}
}

func TestRationalArithmeticEntryDivEndGuardAndOwningRetry(t *testing.T) {
	m, _, _ := rawArithmeticGuardOwner(t)
	r := &m.rawArithmetic
	serviceMust(t, m.z.SetLimbs(&r.div, []uint32{17, 3, 1}))
	r.divActive = true
	ref := r.div.cell
	original := m.z.cell(ref)
	rawArithmeticCorrupt(t, m.z, &r.div)
	e := m.rawDivEnd()
	if !errors.Is(e, arena.ErrBounds) || !r.divActive || !r.divEnd || r.divCursor != 0 || r.div.cell != ref {
		t.Fatal("source end reached real physical refusal", r, e)
	}
	qopsAuditLiveOwnership(t, m, nil)
	if e = m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal(e)
	}
	e = m.CleanupRawArithmetic()
	if !errors.Is(e, arena.ErrBounds) || !r.divActive || r.divCursor != 0 {
		t.Fatal("owning refusal retains actual root", r, e)
	}
	qopsAuditLiveOwnership(t, m, nil)
	// Restore only the deliberately malformed requested-size fixture, never
	// source arithmetic fields or counters, then retry actual owning cleanup.
	c := m.z.cell(ref)
	c.backing.requested = original.backing.requested
	m.z.store(ref, c)
	serviceMust(t, m.CleanupRawArithmetic())
	if r.pending() {
		t.Fatal("owning continuation retained local", r)
	}
	qopsAuditLiveOwnership(t, m, nil)
}
