// Licensed under the MIT license.
// Added source-action coverage: mpq.cpp:186-276 and mpq.h:140-145,324-332,443-480.
package mpnum

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func scientificRawPair(t *testing.T, m *RationalManager, a *Rational) [2]string {
	t.Helper()
	component := func(n *Int) string {
		if n.cell == m.z.failedRef {
			if !errors.Is(m.z.check(n), arena.ErrInvalidSlot) {
				t.Fatal("failed owning descriptor was readable")
			}
			return "failed"
		}
		return roundPublicationDecode(t, m.z, n).String()
	}
	return [2]string{component(&a.num), component(&a.den)}
}

func TestScientificBoundaryZeroDenominatorContinuations(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, initial := range []int32{0, 1, -1} {
			for _, tc := range []struct {
				text string
				n    string
				zero bool
			}{{"2e2", "1", false}, {"-2e-2", "-1", false}, {"0e2", "0", true}, {"0e-2", "0", true}, {"2e0", "1", false}} {
				m, _ := rationalManager(t, sync)
				a := NewRational()
				n, d := NewInt(initial), NewInt(0)
				e := m.SetParts(&a, &n, &d)
				if initial == 0 {
					if !errors.Is(e, ErrDivideByZero) {
						t.Fatal("source-published zero/zero setup", e)
					}
				} else {
					qopsMust(t, e)
				}
				e = m.SetString(&a, tc.text)
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &n, &d})
				if tc.zero {
					if !errors.Is(e, ErrDivideByZero) {
						t.Fatalf("%v %d %q: actual normalize zero divide, got %v", sync, initial, tc.text, e)
					}
				} else if e != nil {
					t.Fatalf("%v %d %q: finite source continuation: %v", sync, initial, tc.text, e)
				}
				if got := scientificRawPair(t, m, &a); got != [2]string{tc.n, "0"} {
					t.Fatal("caller normalization/delayed sign", tc.text, got)
				}
				if _, e := m.String(&a); !errors.Is(e, ErrUninitializedRational) {
					t.Fatal("ordinary reader remains guarded", e)
				}
				qopsMust(t, m.Del(&a))
			}
		}
	}
}

func TestScientificBoundaryPowerRefusalPrecedesRawArithmetic(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, text := range []string{"-2e40", "-2e-40", "-2e2147483648", "-2e-2147483648"} {
			m, budget := rationalPublicationDirectOwner(t, sync)
			a := Rational{num: NewInt(17), den: NewInt(0)}
			budget.FailAfter(0)
			e := m.SetString(&a, text)
			qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den})
			if !numericPhysicalReservationRefusal(e) || scientificRawPair(t, m, &a) != [2]string{"2", "0"} {
				t.Fatal("private Power acts before raw arithmetic and delayed sign", sync, text, a, e)
			}
			budget.FailAfter(-1)
			qopsMust(t, m.Del(&a))
		}
	}
}

func scientificSmallModel(a, b Rational, divide bool) ([2]string, bool) {
	n, d := big.NewInt(int64(a.num.val)), big.NewInt(int64(a.den.val))
	bn, bd := big.NewInt(int64(b.num.val)), big.NewInt(int64(b.den.val))
	if divide {
		n.Mul(n, bd)
		d.Mul(d, bn)
		if d.Sign() < 0 {
			n.Neg(n)
			d.Neg(d)
		}
	} else {
		n.Mul(n, bn)
		d.Mul(d, bd)
	}
	g := new(big.Int).GCD(nil, nil, n, d)
	if g.Sign() == 0 {
		return [2]string{n.String(), d.String()}, true
	}
	n.Quo(n, g)
	d.Quo(d, g)
	return [2]string{n.String(), d.String()}, false
}

func TestScientificBoundaryPublicMulDivRawAliases(t *testing.T) {
	pairs := [][2]int32{{1, 0}, {-2, 0}, {0, 0}, {0, 1}, {2, 3}, {-3, -2}}
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for _, an := range pairs {
			for _, bn := range pairs {
				for _, same := range []bool{false, true} {
					for _, divide := range []bool{false, true} {
						for alias := 0; alias < 3; alias++ {
							a := Rational{num: NewInt(an[0]), den: NewInt(an[1])}
							b := Rational{num: NewInt(bn[0]), den: NewInt(bn[1])}
							out := Rational{num: NewInt(17), den: NewInt(0)}
							right := &b
							if same {
								right = &a
							}
							want, zero := scientificSmallModel(a, *right, divide)
							dst := []*Rational{&out, &a, right}[alias]
							var e error
							if divide {
								e = m.Div(&a, right, dst)
							} else {
								e = m.Mul(&a, right, dst)
							}
							qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
							if (zero && !errors.Is(e, ErrDivideByZero)) || (!zero && e != nil) || scientificRawPair(t, m, dst) != want {
								t.Fatal("literal source raw action model", sync, an, bn, same, divide, alias, want, *dst, e)
							}
							for _, v := range []*Rational{&a, &b, &out} {
								qopsMust(t, m.Del(v))
							}
						}
					}
				}
			}
		}
	}
}

func TestScientificBoundaryFirstRawGrowthRefusal(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, divide := range []bool{false, true} {
			for alias := 0; alias < 3; alias++ {
				m, budget := rationalPublicationDirectOwner(t, sync)
				a, b, out := NewRational(), NewRational(), NewRational()
				an := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 2100), big.NewInt(7))
				bn := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 2000), big.NewInt(3))
				setIntegerOracle(t, m.z, &a.num, an)
				a.den = NewInt(0)
				setIntegerOracle(t, m.z, &b.num, bn)
				b.den = NewInt(7)
				out.num, out.den = NewInt(17), NewInt(13)
				dst := []*Rational{&out, &a, &b}[alias]
				beforeA, beforeB, beforeOut := a, b, out
				budget.FailAfter(0)
				var e error
				if divide {
					e = m.Div(&a, &b, dst)
				} else {
					e = m.Mul(&a, &b, dst)
				}
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
				if !numericPhysicalReservationRefusal(e) || a != beforeA || b != beforeB || out != beforeOut || m.z.tmp[0] != 0 {
					t.Fatal("first real scratch growth after source free, before output action", sync, divide, alias, e)
				}
				budget.FailAfter(-1)
				for _, v := range []*Rational{&a, &b, &out} {
					qopsMust(t, m.Del(v))
				}
			}
		}
	}
}

func TestScientificBoundaryPublicMulDivStillRejectsInvalidHandles(t *testing.T) {
	m, _ := rationalManager(t, false)
	a, b, out := NewRational(), NewRational(), NewRational()
	setIntegerOracle(t, m.z, &a.num, new(big.Int).Lsh(big.NewInt(1), 200))
	stale := a
	qopsMust(t, m.Del(&a))
	for _, divide := range []bool{false, true} {
		var e error
		if divide {
			e = m.Div(&stale, &b, &out)
		} else {
			e = m.Mul(&stale, &b, &out)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
		if !errors.Is(e, arena.ErrInvalidSlot) || out != NewRational() {
			t.Fatal("ordinary Integer handle guard retained", divide, out, e)
		}
	}
}
