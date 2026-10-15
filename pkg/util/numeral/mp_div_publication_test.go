// Added source-action/profile tests for mpq.h division/reciprocal methods.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package numeral

import (
	"errors"
	"math/big"
	"testing"
)

func divPublicationAudit(t *testing.T, m *RationalManager, values ...*Rational) {
	t.Helper()
	roots := make([]*Int, 0, len(values)*2)
	for _, v := range values {
		roots = append(roots, &v.Num, &v.Den)
	}
	qopsAuditLiveOwnership(t, m, roots)
}
func TestDivisionPublicationInverseZeroProfileAndCleanup(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			m, _ := rationalManager(t, sync)
			a, out := NewRational(), NewRational()
			qopsMust(t, m.SetInt64(&out, 71))
			dst := &out
			if alias {
				dst = &a
			}
			e := m.Inv(&a, dst)
			divPublicationAudit(t, m, &a, &out)
			if modularSourceDebug {
				if !errors.Is(e, ErrIntegerAssertion) || dst.Num != NewInt(0) || dst.Den != NewInt(1) {
					t.Fatal("Copy before source inverse assertion", alias, e, *dst)
				}
			} else {
				qopsMust(t, e)
				if dst.Num != NewInt(1) || dst.Den != NewInt(0) {
					t.Fatal("finite source zero inverse", *dst)
				}
				if _, e = m.String(dst); !errors.Is(e, ErrUninitializedRational) {
					t.Fatal("ordinary invalid-denominator guard", e)
				}
				qopsMust(t, m.Reset(dst))
				if *dst != NewRational() {
					t.Fatal("explicit reset action", *dst)
				}
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
		}
	}
}
func TestDivisionPublicationZeroDivisorFiniteActions(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, op := range []int{0, 1, 3} {
			for _, n := range []int64{-2, 0, 2} {
				m, _ := rationalManager(t, sync)
				a, b, out := NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetInt64(&a, n))
				qopsMust(t, m.SetInt64(&out, 71))
				z := NewInt(0)
				num := NewInt(int32(n))
				var e error
				switch op {
				case 0:
					e = m.Div(&a, &b, &out)
				case 1:
					e = m.DivInt(&a, &z, &out)
				case 3:
					e = m.RatDiv(&num, &z, &out)
				}
				divPublicationAudit(t, m, &a, &b, &out)
				if n == 0 {
					if !errors.Is(e, ErrDivideByZero) || out.Num != NewInt(0) || out.Den != NewInt(0) {
						t.Fatal("actual 0/0 unsafe action after component publication", sync, op, e, out)
					}
				} else {
					qopsMust(t, e)
					want := int32(1)
					if n < 0 {
						want = -1
					}
					if out.Num != NewInt(want) || out.Den != NewInt(0) {
						t.Fatal("finite source n/0", sync, op, e, out)
					}
				}
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
			}
		}
	}
}
func TestDivisionPublicationCallerFailurePrefixes(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, op := range []int{0, 1, 2, 4} {
			for alias := 0; alias < 3; alias++ {
				changed, success := false, false
				for fault := int64(0); fault < 256; fault++ {
					m, budget := rationalManager(t, sync)
					a, b, out := NewRational(), NewRational(), NewRational()
					qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981/1180591620717411303427"))
					qopsMust(t, m.SetString(&b, "18446744073709551617/4294967297"))
					qopsMust(t, m.SetInt64(&out, 71))
					dst := []*Rational{&out, &a, &b}[alias]
					beforeNum := integerString(t, m.Z, &dst.Num)
					beforeDen := integerString(t, m.Z, &dst.Den)
					an, _ := new(big.Int).SetString(integerString(t, m.Z, &a.Num), 10)
					ad, _ := new(big.Int).SetString(integerString(t, m.Z, &a.Den), 10)
					bn, _ := new(big.Int).SetString(integerString(t, m.Z, &b.Num), 10)
					bd, _ := new(big.Int).SetString(integerString(t, m.Z, &b.Den), 10)
					dn, _ := new(big.Int).SetString(beforeNum, 10)
					dd, _ := new(big.Int).SetString(beforeDen, 10)
					budget.FailAfter(fault)
					var e error
					switch op {
					case 0:
						e = m.Div(&a, &b, dst)
					case 1:
						e = m.Inv(&a, dst)
					case 2:
						e = m.DivInt(&a, &b.Num, dst)
					case 4:
						e = m.RatDiv(&a.Num, &b.Num, dst)
					}
					divPublicationAudit(t, m, &a, &b, &out)
					if e != nil {
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal(sync, op, alias, fault, e)
						}
					} else {
						success = true
					}
					budget.FailAfter(-1)
					if e != nil {
						ns, ds := []string{beforeNum}, []string{beforeDen}
						var pn, pd *big.Int
						switch op {
						case 0:
							pn = new(big.Int).Mul(an, bd)
							pd = new(big.Int).Mul(ad, bn)
						case 1:
							ns = append(ns, an.String(), ad.String())
							ds = append(ds, ad.String(), an.String())
						case 2:
							divisor := bn
							if alias == 2 {
								divisor = an
							}
							pn = new(big.Int).Set(an)
							pd = new(big.Int).Mul(ad, divisor)
						case 3:
							pn = new(big.Int).Set(dn)
							pd = new(big.Int).Mul(dd, bn)
						case 4:
							divisor := bn
							if alias == 2 {
								divisor = an
							}
							pn = new(big.Int).Set(an)
							pd = new(big.Int).Set(divisor)
						}
						if pn != nil {
							ns = append(ns, pn.String())
							ds = append(ds, pd.String())
							normalized := new(big.Rat).SetFrac(pn, pd)
							ns = append(ns, normalized.Num().String())
							ds = append(ds, normalized.Denom().String())
						}
						powerPublicationComponent(t, m.Z, &dst.Num, ns...)
						powerPublicationComponent(t, m.Z, &dst.Den, ds...)
					}
					if e != nil && (m.Z.IsFailureRef(dst.Num.Cell) || m.Z.IsFailureRef(dst.Den.Cell) || integerString(t, m.Z, &dst.Num) != beforeNum || integerString(t, m.Z, &dst.Den) != beforeDen) {
						changed = true
					}
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&b))
					qopsMust(t, m.Del(&out))
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("leak", budget.Used())
					}
					if success {
						break
					}
				}
				if !success || (!changed && op == 0 && alias == 0) {
					t.Fatal("direct caller action before later refusal", sync, op, alias, changed, success)
				}
			}
		}
	}
}

// Three actual registered source ENSURE conditions, with independent expected
// values used only for setup. Repetition over storage policies is not new count.
func TestDivisionPublicationSourceThreeAssertions(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		a, b, want := NewRational(), NewRational(), NewRational()
		qopsMust(t, m.SetFraction(&a, 2, 3))
		qopsMust(t, m.SetFraction(&b, 4, 3))
		qopsMust(t, m.Div(&a, &b, &b))
		qopsMust(t, m.SetFraction(&want, 1, 2))
		eq, e := m.Equal(&b, &want)
		qopsMust(t, e)
		if !eq {
			t.Fatal("tst0 ENSURE")
		}
		for _, n := range []int64{2, -2} {
			qopsMust(t, m.SetInt64(&a, n))
			qopsMust(t, m.SetFraction(&b, n/2, 2))
			qopsMust(t, m.Inv(&a, &a))
			eq, e = m.Equal(&a, &b)
			qopsMust(t, e)
			if !eq {
				t.Fatal("bug1/bug2 ENSURE", n)
			}
		}
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
		qopsMust(t, m.Del(&want))
	}
}
func TestDivisionPublicationFractionalIntegerForwardProfiles(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			m, _ := rationalManager(t, sync)
			a, b, out := NewRational(), NewRational(), NewRational()
			qopsMust(t, m.SetFraction(&a, 5, 2))
			qopsMust(t, m.SetFraction(&b, 3, 2))
			qopsMust(t, m.SetFraction(&out, 7, 9))
			var e error
			switch op {
			case 0:
				e = m.IDiv(&a, &b, &out)
			case 1:
				e = m.MachineIDiv(&a, &b, &out)
			case 2:
				e = m.Rem(&a, &b, &out)
			case 3:
				e = m.Mod(&a, &b, &out)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
			if modularSourceDebug {
				if !errors.Is(e, ErrRationalNotInteger) || out.Num != NewInt(7) || out.Den != NewInt(9) {
					t.Fatal(e)
				}
			} else {
				qopsMust(t, e)
				want := []int32{1, 1, 2, 2}[op]
				qopsEqual(t, m, &out, new(big.Rat).SetInt64(int64(want)))
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&b))
			qopsMust(t, m.Del(&out))
		}
	}
}

func TestDivisionPublicationInverseInPlaceOverload(t *testing.T) {
	m, _ := rationalManager(t, true)
	a := NewRational()
	qopsMust(t, m.SetFraction(&a, -3, 7))
	qopsMust(t, m.InvInPlace(&a))
	divPublicationAudit(t, m, &a)
	qopsEqual(t, m, &a, new(big.Rat).SetFrac64(-7, 3))
	qopsMust(t, m.Del(&a))
}

func TestDivisionPublicationInverseMinNegFailureOrder(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			m, budget := rationalPublicationDirectOwner(t, sync)
			a := Rational{Num: NewInt(-2147483648), Den: NewInt(1)}
			out := NewRational()
			qopsMust(t, m.SetInt64(&out, 71))
			dst := &out
			if alias {
				dst = &a
			}
			// Fill actual free descriptor slots so the promoting source Neg must reserve.
			var bank []Int
			for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
				var v Int
				qopsMust(t, m.Z.SetUint64(&v, 1<<40))
				bank = append(bank, v)
			}
			roots := []*Int{&a.Num, &a.Den, &out.Num, &out.Den}
			for i := range bank {
				roots = append(roots, &bank[i])
			}
			budget.FailAfter(0)
			e := m.Inv(&a, dst)
			qopsAuditLiveOwnership(t, m, roots)
			if !numericPhysicalReservationRefusal(e) || dst.Num != NewInt(-2147483648) || dst.Den != NewInt(1) {
				t.Fatal("Copy then Neg refusal before later Neg/Swap", alias, e, *dst)
			}
			budget.FailAfter(-1)
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
			for i := range bank {
				qopsMust(t, m.Z.Del(&bank[i]))
			}
		}
	}
}

func TestDivisionPublicationEarlierAllocationPrecedesZeroDomain(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, budget := rationalManager(t, sync)
		a, b, out := NewRational(), NewRational(), NewRational()
		qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981"))
		qopsMust(t, m.SetInt64(&out, 71))
		budget.FailAfter(0)
		e := m.Div(&a, &b, &out)
		divPublicationAudit(t, m, &a, &b, &out)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("real growth precedes later n/0 actions", e)
		}
		budget.FailAfter(-1)
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
		qopsMust(t, m.Del(&out))
	}
}

func TestDivisionPublicationComponentAliasSourceActions(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 2; op++ {
			m, _ := rationalManager(t, sync)
			a, out := NewRational(), NewRational()
			qopsMust(t, m.SetFraction(&a, -3, 5))
			qopsMust(t, m.SetFraction(&out, 7, 11))
			var e error
			var wantNum, wantDen string
			switch op {
			case 0:
				e = m.DivInt(&a, &out.Num, &out)
				wantNum, wantDen = "1", "5"
			case 1:
				e = m.DivInt(&a, &out.Den, &out)
				wantNum, wantDen = "-3", "55"
			}
			qopsMust(t, e)
			divPublicationAudit(t, m, &a, &out)
			if integerString(t, m.Z, &out.Num) != wantNum || integerString(t, m.Z, &out.Den) != wantDen {
				t.Fatal("actual divisor reread", sync, op, out)
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
		}
	}
	for _, sync := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			m, _ := rationalManager(t, sync)
			a, out := NewRational(), NewRational()
			qopsMust(t, m.SetFraction(&a, -3, 5))
			qopsMust(t, m.SetFraction(&out, 7, 11))
			var n, d *Int
			var wantNum, wantDen string
			switch op {
			case 0:
				n, d = &a.Num, &out.Num
				wantNum, wantDen = "-1", "-1"
			case 1:
				n, d = &a.Num, &out.Den
				wantNum, wantDen = "-3", "11"
			case 2:
				n, d = &out.Den, &a.Num
				wantNum, wantDen = "11", "-3"
			case 3:
				n, d = &out.Den, &out.Num
				wantNum, wantDen = "1", "1"
			}
			qopsMust(t, m.RatDiv(n, d, &out))
			divPublicationAudit(t, m, &a, &out)
			if integerString(t, m.Z, &out.Num) != wantNum || integerString(t, m.Z, &out.Den) != wantDen {
				t.Fatal("actual ordered Copy and raw normalization", sync, op, out)
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
		}
	}
}

func TestDivisionPublicationRawUnreducedAndDenominatorSigns(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, n := range []int64{-6, 6} {
			for _, d := range []int64{-4, 4} {
				for alias := 0; alias < 3; alias++ {
					m, _ := rationalManager(t, sync)
					a, b, out := NewRational(), NewRational(), NewRational()
					qopsMust(t, m.Z.SetInt64(&a.Num, n))
					qopsMust(t, m.Z.SetInt64(&a.Den, d))
					qopsMust(t, m.Z.SetInt64(&b.Num, 3))
					qopsMust(t, m.Z.SetInt64(&b.Den, -9))
					dst := []*Rational{&out, &a, &b}[alias]
					qopsMust(t, m.Div(&a, &b, dst))
					divPublicationAudit(t, m, &a, &b, &out)
					qopsEqual(t, m, dst, new(big.Rat).Quo(new(big.Rat).SetFrac64(n, d), new(big.Rat).SetFrac64(3, -9)))
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&b))
					qopsMust(t, m.Del(&out))
					fresh, _ := rationalManager(t, sync)
					a, out = NewRational(), NewRational()
					qopsMust(t, fresh.Z.SetInt64(&a.Num, n))
					qopsMust(t, fresh.Z.SetInt64(&a.Den, d))
					qopsMust(t, fresh.Inv(&a, &out))
					divPublicationAudit(t, fresh, &a, &out)
					nn, dd := d, n
					if n < 0 {
						nn = -d
						dd = -n
					}
					if integerString(t, fresh.Z, &out.Num) != big.NewInt(nn).String() || integerString(t, fresh.Z, &out.Den) != big.NewInt(dd).String() {
						t.Fatal("Inv does not normalize raw fields", n, d, out)
					}
					qopsMust(t, fresh.Del(&a))
					qopsMust(t, fresh.Del(&out))
				}
			}
		}
	}
}

func TestDivisionPublicationServiceInvalidResultCopyAndCleanup(t *testing.T) {
	s, budget := serviceNew(t)
	a := serviceValue(t, s, "2")
	zero := serviceValue(t, s, "0")
	out, e := s.Div(&a, &zero)
	facadeAudit(t, s, &a, &zero, &out)
	if e != nil || out.ownerID != s.ownerID || out.Raw.Num != NewInt(1) || out.Raw.Den != NewInt(0) {
		t.Fatal("source value-result Copy preserves finite raw division", e, out)
	}
	if a.Raw.Num != NewInt(2) || a.Raw.Den != NewInt(1) {
		t.Fatal("value-result division retains its input", a.Raw)
	}
	serviceMust(t, s.Del(&out))
	serviceMust(t, s.DivAssign(&a, &zero))
	facadeAudit(t, s, &a, &zero)
	if a.Raw.Num != NewInt(1) || a.Raw.Den != NewInt(0) {
		t.Fatal("source compound field publication", a.Raw)
	}
	if _, e = s.String(&a); !errors.Is(e, ErrUninitializedRational) {
		t.Fatal(e)
	}
	serviceMust(t, s.Del(&a))
	serviceMust(t, s.Del(&zero))
	serviceMust(t, s.Close())
	if budget.Used() != 0 {
		t.Fatal("invalid owned pair leaked", budget.Used())
	}
}

func TestDivisionPublicationIntegerForwardRawSignsAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for x := int32(-7); x <= 7; x++ {
			for y := int32(-3); y <= 3; y++ {
				if y == 0 {
					continue
				}
				for op := 0; op < 4; op++ {
					for alias := 0; alias < 4; alias++ {
						a := Rational{Num: NewInt(x), Den: NewInt(-2)}
						b := Rational{Num: NewInt(y), Den: NewInt(3)}
						out := NewRational()
						out.Num = NewInt(71)
						left, right := &a, &b
						dst := &out
						if alias == 1 {
							dst = left
						}
						if alias == 2 {
							dst = right
						}
						if alias == 3 {
							right = left
							dst = left
						}
						var e error
						switch op {
						case 0:
							e = m.IDiv(left, right, dst)
						case 1:
							e = m.MachineIDiv(left, right, dst)
						case 2:
							e = m.Rem(left, right, dst)
						case 3:
							e = m.Mod(left, right, dst)
						}
						qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den, &out.Num, &out.Den})
						if modularSourceDebug {
							if !errors.Is(e, ErrRationalNotInteger) {
								t.Fatal(e)
							}
							continue
						}
						yy := y
						if alias == 3 {
							yy = x
						}
						if yy == 0 {
							if !errors.Is(e, ErrDivideByZero) {
								t.Fatal(e)
							}
							continue
						}
						qopsMust(t, e)
						quot, rem := x/yy, x%yy
						want := quot
						if op == 0 && x < 0 && rem != 0 {
							sign := yy
							if alias == 2 {
								sign = quot
							}
							if sign < 0 {
								want++
							} else {
								want--
							}
						}
						if op >= 2 {
							want = rem
							if op == 3 && rem < 0 {
								divisor := yy
								if alias == 2 {
									divisor = rem
								}
								if divisor > 0 {
									want += divisor
								} else {
									want -= divisor
								}
							}
						}
						if dst.Den != NewInt(1) {
							t.Fatal("den reset")
						}
						if dst.Num != NewInt(want) {
							t.Fatal("source finite raw/sign/alias action", x, y, op, alias, dst.Num, want)
						}
					}
				}
			}
		}
	}

}

func TestDivisionPublicationWarmStorage(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, budget := rationalManager(t, sync)
		a, b, out := NewRational(), NewRational(), NewRational()
		qopsMust(t, m.SetString(&a, "18446744073709551617/4294967297"))
		qopsMust(t, m.SetString(&b, "4294967297/65537"))
		run := func() {
			qopsMust(t, m.Div(&a, &b, &out))
			qopsMust(t, m.Inv(&a, &out))
			qopsMust(t, m.DivInt(&a, &b.Num, &out))
			qopsMust(t, m.RatDiv(&a.Num, &b.Num, &out))
		}
		for i := 0; i < 10; i++ {
			run()
		}
		before := numericWarmProfileSnapshot(t, m.Z, budget)
		n := testing.AllocsPerRun(50, run)
		numericWarmProfileAudit(t, m.Z, budget, before, n, SourceExtendedGCDDebug)
		divPublicationAudit(t, m, &a, &b, &out)
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
		qopsMust(t, m.Del(&out))
	}
}

func TestDivisionPublicationZeroDivisorOutputAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, n := range []int64{-2, 0, 2} {
			for alias := 0; alias < 4; alias++ {
				if alias == 3 && n != 0 {
					continue
				}
				m, _ := rationalManager(t, sync)
				a, b, out := NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetInt64(&a, n))
				qopsMust(t, m.SetInt64(&out, 71))
				left, right, dst := &a, &b, &out
				if alias == 1 {
					dst = left
				}
				if alias == 2 {
					dst = right
				}
				if alias == 3 {
					left, right, dst = &b, &b, &b
				}
				e := m.Div(left, right, dst)
				divPublicationAudit(t, m, &a, &b, &out)
				if n == 0 {
					if !errors.Is(e, ErrDivideByZero) || dst.Num != NewInt(0) || dst.Den != NewInt(0) {
						t.Fatal(sync, n, alias, e, *dst)
					}
				} else {
					qopsMust(t, e)
					want := int32(1)
					if n < 0 {
						want = -1
					}
					if dst.Num != NewInt(want) || dst.Den != NewInt(0) {
						t.Fatal(sync, n, alias, *dst)
					}
				}
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
			}
		}
	}
}

func TestDivisionPublicationInverseCopyFailureBeforeDebugAssertion(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, budget := rationalManager(t, sync)
		a, out := NewRational(), NewRational()
		qopsMust(t, m.Z.SetUint64(&a.Den, 1<<40))
		qopsMust(t, m.SetInt64(&out, 71))
		var bank []Int
		for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
			var v Int
			qopsMust(t, m.Z.Copy(&v, &a.Den))
			bank = append(bank, v)
		}
		roots := []*Int{&a.Num, &a.Den, &out.Num, &out.Den}
		for i := range bank {
			roots = append(roots, &bank[i])
		}
		budget.FailAfter(0)
		e := m.Inv(&a, &out)
		qopsAuditLiveOwnership(t, m, roots)
		if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(0) || out.Den.Cell != m.Z.FailedNullRef {
			t.Fatal("real denominator Copy precedes zero assertion/Swap", e, out)
		}
		budget.FailAfter(-1)
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&out))
		for i := range bank {
			qopsMust(t, m.Z.Del(&bank[i]))
		}
	}
}

func TestDivisionPublicationInverseTwoMinNegActions(t *testing.T) {
	for _, sync := range []bool{false, true} {
		success, failed := false, false
		for fault := int64(0); fault < 64; fault++ {
			m, budget := rationalManager(t, sync)
			a := Rational{Num: NewInt(-2147483648), Den: NewInt(-2147483648)}
			out := NewRational()
			qopsMust(t, m.SetInt64(&out, 71))
			var bank []Int
			for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
				var v Int
				qopsMust(t, m.Z.SetUint64(&v, 1<<40))
				bank = append(bank, v)
			}
			roots := []*Int{&a.Num, &a.Den, &out.Num, &out.Den}
			for i := range bank {
				roots = append(roots, &bank[i])
			}
			budget.FailAfter(fault)
			e := m.Inv(&a, &out)
			qopsAuditLiveOwnership(t, m, roots)
			if e != nil {
				failed = true
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
			} else {
				success = true
			}
			budget.FailAfter(-1)
			num := powerPublicationComponent(t, m.Z, &out.Num, "-2147483648", "2147483648")
			den := powerPublicationComponent(t, m.Z, &out.Den, "-2147483648", "2147483648")
			if num == "-2147483648" && den != "-2147483648" {
				t.Fatal("denominator Neg preceded numerator Neg")
			}
			if success && (num != "2147483648" || den != "2147483648") {
				t.Fatal("raw reciprocal never normalizes 1/1 fields", out)
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
			for i := range bank {
				qopsMust(t, m.Z.Del(&bank[i]))
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("root refund", budget.Used())
			}
			if success {
				break
			}
		}
		if !failed || !success {
			t.Fatal(sync, failed, success)
		}
	}
}
