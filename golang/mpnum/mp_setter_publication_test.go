// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpq.h paired setter branch and caller publication actions.
package mpnum

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func TestSetterPublicationPartsCrossedAliasBranch(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, f := range []struct{ oldN, oldD, wantN, wantD int32 }{{5, -3, -1, -1}, {-5, -3, 1, 1}, {5, 3, 1, 1}, {-5, 3, -1, -1}} {
			m, budget := rationalManager(t, synch)
			a := Rational{num: NewInt(f.oldN), den: NewInt(f.oldD)}
			e := m.SetParts(&a, &a.den, &a.num)
			parserPublicationAudit(t, m, &a.num, &a.den)
			if e != nil || roundPublicationDecode(t, m.z, &a.num).Cmp(big.NewInt(int64(f.wantN))) != 0 || roundPublicationDecode(t, m.z, &a.den).Cmp(big.NewInt(int64(f.wantD))) != 0 {
				t.Errorf("branch before writes and actual later input reread: synch=%v old=%d/%d raw=%+v err=%v", synch, f.oldN, f.oldD, a, e)
			}
			qopsMust(t, m.Del(&a))
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
	}
}
func TestSetterPublicationPartsZeroDenominator(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, n := range []int32{2, -2, 0} {
			m, budget := rationalManager(t, synch)
			a := Rational{num: NewInt(17), den: NewInt(13)}
			num, den := NewInt(n), NewInt(0)
			e := m.SetParts(&a, &num, &den)
			parserPublicationAudit(t, m, &a.num, &a.den, &num, &den)
			want := int64(0)
			if n > 0 {
				want = 1
			}
			if n < 0 {
				want = -1
			}
			if n == 0 {
				if !errors.Is(e, ErrDivideByZero) {
					t.Errorf("actual normalize divide-by-zero: %v", e)
				}
			} else if e != nil {
				t.Errorf("finite raw source nonzero/0: %v", e)
			}
			if roundPublicationDecode(t, m.z, &a.num).Cmp(big.NewInt(want)) != 0 || a.den != NewInt(0) {
				t.Errorf("source actual normalize publication: n=%d raw=%+v", n, a)
			}
			if _, e = m.IsInt(&a); !errors.Is(e, ErrUninitializedRational) {
				t.Errorf("ordinary invalid read remains guarded: %v", e)
			}
			qopsMust(t, m.Del(&a))
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
	}
}
func TestSetterPublicationScalarNumeratorBeforeRefusal(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a := Rational{num: NewInt(17), den: NewInt(19)}
		bank := roundPublicationFillCells(t, m.z)
		roots := []*Int{&a.num, &a.den}
		for i := range bank {
			roots = append(roots, bank[i])
		}
		budget.FailAfter(0)
		e := m.SetFraction(&a, 1, math.MaxUint64)
		parserPublicationAudit(t, m, roots...)
		var failure memory.Failure
		if !errors.Is(e, arena.ErrInjectedFailure) && (!errors.As(e, &failure) || failure.Kind != memory.FailureMemout || !m.z.memory.State().OutOfMemory) {
			t.Fatal("actual armed growth error", e)
		}
		if a.num != NewInt(1) || a.den != NewInt(19) {
			t.Errorf("NUM source setter precedes refused DEN allocation: synch=%v raw=%+v", synch, a)
		}
		budget.FailAfter(-1)
		qopsMust(t, m.Del(&a))
		for i := range bank {
			qopsMust(t, m.z.Del(bank[i]))
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}
}
