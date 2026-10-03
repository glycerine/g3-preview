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
		budget := arena.NewBudget(1 << 20)
		m, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 1, Budget: budget}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: budget}}}, Synchronized: synch})
		qopsMust(t, e)
		t.Cleanup(func() {
			m.Close()
			if budget.Used() != 0 {
				t.Error(budget.Used())
			}
		})
		a := Rational{num: NewInt(17), den: NewInt(19)}
		bank := roundPublicationFillCells(t, m.z)
		roots := []*Int{&a.num, &a.den}
		for i := range bank {
			roots = append(roots, bank[i])
		}
		budget.FailAfter(0)
		e = m.SetFraction(&a, 1, math.MaxUint64)
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

func TestSetterPublicationTypedScalarRoutes(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		a := NewRational()
		for _, f := range []struct {
			n, d int32
			want string
		}{{2, 3, "2/3"}, {-2, 3, "-2/3"}, {2, -3, "-2/3"}, {-2, -3, "2/3"}, {math.MinInt32, -1, "2147483648"}} {
			qopsMust(t, m.SetFractionSigned(&a, f.n, f.d))
			parserPublicationAudit(t, m, &a.num, &a.den)
			if rationalString(t, m, &a) != f.want {
				t.Fatal(f, a)
			}
		}
		qopsMust(t, m.SetInt32(&a, math.MinInt32))
		if a.num != NewInt(math.MinInt32) || a.den != NewInt(1) {
			t.Fatal("signed32 source shape", a)
		}
		qopsMust(t, m.SetUint32(&a, math.MaxInt32))
		if a.num != NewInt(math.MaxInt32) {
			t.Fatal("unsigned32 inclusive shape", a)
		}
		qopsMust(t, m.SetUint64(&a, math.MaxInt32))
		if a.num.IsSmall() {
			t.Fatal("unsigned64 strict source threshold", a)
		}
		qopsMust(t, m.Del(&a))
		m.Close()
		if budget.Used() != 0 {
			t.Fatal(budget.Used())
		}
	}
}
func TestSetterPublicationPartsAllInputIdentities(t *testing.T) {
	for _, synch := range []bool{false, true} {
		for _, sign := range []int64{-1, 1} {
			for ni := 0; ni < 3; ni++ {
				for di := 0; di < 3; di++ {
					m, budget := rationalManager(t, synch)
					a := Rational{num: NewInt(5), den: NewInt(-3)}
					n, d := NewInt(int32(7*sign)), NewInt(int32(-11*sign))
					nums := []*Int{&n, &a.num, &a.den}
					dens := []*Int{&d, &a.num, &a.den}
					// Independent source action simulation captures the branch first, then
					// reads actual aliased NUM/DEN references at their Copy statements.
					rawN, rawD := big.NewInt(5), big.NewInt(-3)
					read := func(index int, external *Int) *big.Int {
						switch index {
						case 1:
							return new(big.Int).Set(rawN)
						case 2:
							return new(big.Int).Set(rawD)
						default:
							return big.NewInt(int64(external.val))
						}
					}
					negative := read(di, &d).Sign() < 0
					rawN = read(ni, &n)
					rawD = read(di, &d)
					if negative {
						rawN.Neg(rawN)
						rawD.Neg(rawD)
					}
					g := new(big.Int).GCD(nil, nil, rawN, rawD)
					rawN.Quo(rawN, g)
					rawD.Quo(rawD, g)
					e := m.SetParts(&a, nums[ni], dens[di])
					parserPublicationAudit(t, m, &a.num, &a.den, &n, &d)
					if e != nil || roundPublicationDecode(t, m.z, &a.num).Cmp(rawN) != 0 || roundPublicationDecode(t, m.z, &a.den).Cmp(rawD) != 0 {
						t.Errorf("source input identities %v %d %d %d: raw=%+v want=%v/%v err=%v", synch, sign, ni, di, a, rawN, rawD, e)
					}
					qopsMust(t, m.Del(&a))
					m.Close()
					if budget.Used() != 0 {
						t.Fatal(budget.Used())
					}
				}
			}
		}
	}
}
