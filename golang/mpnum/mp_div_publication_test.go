// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Added source-action/profile tests for mpq.h division/reciprocal methods.
package mpnum

import (
	"errors"
	"math/big"
	"testing"
)

func divPublicationAudit(t *testing.T, m *RationalManager, values ...*Rational) {
	t.Helper()
	roots := make([]*Int, 0, len(values)*2)
	for _, v := range values {
		roots = append(roots, &v.num, &v.den)
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
				if !errors.Is(e, ErrIntegerAssertion) || dst.num != NewInt(0) || dst.den != NewInt(1) {
					t.Fatal("Copy before source inverse assertion", alias, e, *dst)
				}
			} else {
				qopsMust(t, e)
				if dst.num != NewInt(1) || dst.den != NewInt(0) {
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
		for op := 0; op < 4; op++ {
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
				case 2:
					qopsMust(t, m.Copy(&out, &a))
					e = m.AccDiv(&out, &z)
				case 3:
					e = m.RatDiv(&num, &z, &out)
				}
				divPublicationAudit(t, m, &a, &b, &out)
				if n == 0 {
					if !errors.Is(e, ErrDivideByZero) || out.num != NewInt(0) || out.den != NewInt(0) {
						t.Fatal("actual 0/0 unsafe action after component publication", sync, op, e, out)
					}
				} else {
					qopsMust(t, e)
					want := int32(1)
					if n < 0 {
						want = -1
					}
					if out.num != NewInt(want) || out.den != NewInt(0) {
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
		for op := 0; op < 5; op++ {
			for alias := 0; alias < 3; alias++ {
				changed, success := false, false
				for fault := int64(0); fault < 256; fault++ {
					m, budget := rationalManager(t, sync)
					a, b, out := NewRational(), NewRational(), NewRational()
					qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981/1180591620717411303427"))
					qopsMust(t, m.SetString(&b, "18446744073709551617/4294967297"))
					qopsMust(t, m.SetInt64(&out, 71))
					dst := []*Rational{&out, &a, &b}[alias]
					beforeNum := integerString(t, m.z, &dst.num)
					beforeDen := integerString(t, m.z, &dst.den)
					an, _ := new(big.Int).SetString(integerString(t, m.z, &a.num), 10)
					ad, _ := new(big.Int).SetString(integerString(t, m.z, &a.den), 10)
					bn, _ := new(big.Int).SetString(integerString(t, m.z, &b.num), 10)
					bd, _ := new(big.Int).SetString(integerString(t, m.z, &b.den), 10)
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
						e = m.DivInt(&a, &b.num, dst)
					case 3:
						e = m.AccDiv(dst, &b.num)
					case 4:
						e = m.RatDiv(&a.num, &b.num, dst)
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
						powerPublicationComponent(t, m.z, &dst.num, ns...)
						powerPublicationComponent(t, m.z, &dst.den, ds...)
					}
					if e != nil && (dst.num.cell == m.z.failedRef || dst.den.cell == m.z.failedRef || integerString(t, m.z, &dst.num) != beforeNum || integerString(t, m.z, &dst.den) != beforeDen) {
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
			for _, integerOut := range []bool{false, true} {
				m, _ := rationalManager(t, sync)
				a, b, out := NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetFraction(&a, 5, 2))
				qopsMust(t, m.SetFraction(&b, 3, 2))
				qopsMust(t, m.SetFraction(&out, 7, 9))
				z := NewInt(71)
				var e error
				if integerOut {
					switch op {
					case 0:
						e = m.IDivToInt(&a, &b, &z)
					case 1:
						e = m.MachineIDivToInt(&a, &b, &z)
					case 2:
						e = m.RemToInt(&a, &b, &z)
					case 3:
						e = m.ModToInt(&a, &b, &z)
					}
				} else {
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
				}
				qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den, &z})
				if modularSourceDebug {
					if !errors.Is(e, ErrRationalNotInteger) || z != NewInt(71) || out.num != NewInt(7) || out.den != NewInt(9) {
						t.Fatal(e)
					}
				} else {
					qopsMust(t, e)
					want := []int32{1, 1, 2, 2}[op]
					if integerOut {
						if z != NewInt(want) {
							t.Fatal(z)
						}
					} else {
						qopsEqual(t, m, &out, new(big.Rat).SetInt64(int64(want)))
					}
				}
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
				qopsMust(t, m.z.Del(&z))
			}
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
			a := Rational{num: NewInt(-2147483648), den: NewInt(1)}
			out := NewRational()
			qopsMust(t, m.SetInt64(&out, 71))
			dst := &out
			if alias {
				dst = &a
			}
			// Fill actual free descriptor slots so the promoting source Neg must reserve.
			var bank []Int
			for m.z.cells.Stats().Live+m.z.cells.Stats().Retired < m.z.cells.Stats().Slots {
				var v Int
				qopsMust(t, m.z.SetUint64(&v, 1<<40))
				bank = append(bank, v)
			}
			roots := []*Int{&a.num, &a.den, &out.num, &out.den}
			for i := range bank {
				roots = append(roots, &bank[i])
			}
			budget.FailAfter(0)
			e := m.Inv(&a, dst)
			qopsAuditLiveOwnership(t, m, roots)
			if !numericPhysicalReservationRefusal(e) || dst.num != NewInt(-2147483648) || dst.den != NewInt(1) {
				t.Fatal("Copy then Neg refusal before later Neg/Swap", alias, e, *dst)
			}
			budget.FailAfter(-1)
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
			for i := range bank {
				qopsMust(t, m.z.Del(&bank[i]))
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
		for op := 0; op < 4; op++ {
			m, _ := rationalManager(t, sync)
			a, out := NewRational(), NewRational()
			qopsMust(t, m.SetFraction(&a, -3, 5))
			qopsMust(t, m.SetFraction(&out, 7, 11))
			var e error
			var wantNum, wantDen string
			switch op {
			case 0:
				e = m.DivInt(&a, &out.num, &out)
				wantNum, wantDen = "1", "5"
			case 1:
				e = m.DivInt(&a, &out.den, &out)
				wantNum, wantDen = "-3", "55"
			case 2:
				e = m.AccDiv(&a, &a.num)
				out = a
				a = NewRational()
				wantNum, wantDen = "1", "5"
			case 3:
				e = m.AccDiv(&a, &a.den)
				out = a
				a = NewRational()
				wantNum, wantDen = "-3", "25"
			}
			qopsMust(t, e)
			divPublicationAudit(t, m, &a, &out)
			if integerString(t, m.z, &out.num) != wantNum || integerString(t, m.z, &out.den) != wantDen {
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
				n, d = &a.num, &out.num
				wantNum, wantDen = "-1", "-1"
			case 1:
				n, d = &a.num, &out.den
				wantNum, wantDen = "-3", "11"
			case 2:
				n, d = &out.den, &a.num
				wantNum, wantDen = "11", "-3"
			case 3:
				n, d = &out.den, &out.num
				wantNum, wantDen = "1", "1"
			}
			qopsMust(t, m.RatDiv(n, d, &out))
			divPublicationAudit(t, m, &a, &out)
			if integerString(t, m.z, &out.num) != wantNum || integerString(t, m.z, &out.den) != wantDen {
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
					qopsMust(t, m.z.SetInt64(&a.num, n))
					qopsMust(t, m.z.SetInt64(&a.den, d))
					qopsMust(t, m.z.SetInt64(&b.num, 3))
					qopsMust(t, m.z.SetInt64(&b.den, -9))
					dst := []*Rational{&out, &a, &b}[alias]
					qopsMust(t, m.Div(&a, &b, dst))
					divPublicationAudit(t, m, &a, &b, &out)
					qopsEqual(t, m, dst, new(big.Rat).Quo(new(big.Rat).SetFrac64(n, d), new(big.Rat).SetFrac64(3, -9)))
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&b))
					qopsMust(t, m.Del(&out))
					fresh, _ := rationalManager(t, sync)
					a, out = NewRational(), NewRational()
					qopsMust(t, fresh.z.SetInt64(&a.num, n))
					qopsMust(t, fresh.z.SetInt64(&a.den, d))
					qopsMust(t, fresh.Inv(&a, &out))
					divPublicationAudit(t, fresh, &a, &out)
					nn, dd := d, n
					if n < 0 {
						nn = -d
						dd = -n
					}
					if integerString(t, fresh.z, &out.num) != big.NewInt(nn).String() || integerString(t, fresh.z, &out.den) != big.NewInt(dd).String() {
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
	if !errors.Is(e, ErrUninitializedRational) || out.ownerID != 0 {
		t.Fatal("Go value-result Copy boundary after finite raw division", e, out)
	}
	serviceMust(t, s.DivAssign(&a, &zero))
	facadeAudit(t, s, &a, &zero)
	if a.raw.num != NewInt(1) || a.raw.den != NewInt(0) {
		t.Fatal("source compound field publication", a.raw)
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
						for _, intOut := range []bool{false, true} {
							a := Rational{num: NewInt(x), den: NewInt(-2)}
							b := Rational{num: NewInt(y), den: NewInt(3)}
							out := NewRational()
							out.num = NewInt(71)
							z := NewInt(71)
							left, right := &a, &b
							dst := &out
							zdst := &z
							if alias == 1 {
								dst = left
								zdst = &a.num
							}
							if alias == 2 {
								dst = right
								zdst = &b.num
							}
							if alias == 3 {
								right = left
								dst = left
								zdst = &a.num
							}
							var e error
							if intOut {
								switch op {
								case 0:
									e = m.IDivToInt(left, right, zdst)
								case 1:
									e = m.MachineIDivToInt(left, right, zdst)
								case 2:
									e = m.RemToInt(left, right, zdst)
								case 3:
									e = m.ModToInt(left, right, zdst)
								}
							} else {
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
							}
							qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den, &z})
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
							got := zdst
							if !intOut {
								got = &dst.num
								if dst.den != NewInt(1) {
									t.Fatal("den reset")
								}
							}
							if *got != NewInt(want) {
								t.Fatal("source finite raw/sign/alias action", x, y, op, alias, intOut, *got, want)
							}
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
			qopsMust(t, m.DivInt(&a, &b.num, &out))
			qopsMust(t, m.AccDiv(&out, &b.num))
			qopsMust(t, m.RatDiv(&a.num, &b.num, &out))
		}
		for i := 0; i < 10; i++ {
			run()
		}
		before := numericWarmProfileSnapshot(t, m.z, budget)
		n := testing.AllocsPerRun(50, run)
		numericWarmProfileAudit(t, m.z, budget, before, n, sourceExtendedGCDDebug)
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
					if !errors.Is(e, ErrDivideByZero) || dst.num != NewInt(0) || dst.den != NewInt(0) {
						t.Fatal(sync, n, alias, e, *dst)
					}
				} else {
					qopsMust(t, e)
					want := int32(1)
					if n < 0 {
						want = -1
					}
					if dst.num != NewInt(want) || dst.den != NewInt(0) {
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
		qopsMust(t, m.z.SetUint64(&a.den, 1<<40))
		qopsMust(t, m.SetInt64(&out, 71))
		var bank []Int
		for m.z.cells.Stats().Live+m.z.cells.Stats().Retired < m.z.cells.Stats().Slots {
			var v Int
			qopsMust(t, m.z.Copy(&v, &a.den))
			bank = append(bank, v)
		}
		roots := []*Int{&a.num, &a.den, &out.num, &out.den}
		for i := range bank {
			roots = append(roots, &bank[i])
		}
		budget.FailAfter(0)
		e := m.Inv(&a, &out)
		qopsAuditLiveOwnership(t, m, roots)
		if !numericPhysicalReservationRefusal(e) || out.num != NewInt(0) || out.den.cell != m.z.failedRef {
			t.Fatal("real denominator Copy precedes zero assertion/Swap", e, out)
		}
		budget.FailAfter(-1)
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&out))
		for i := range bank {
			qopsMust(t, m.z.Del(&bank[i]))
		}
	}
}

func TestDivisionPublicationInverseTwoMinNegActions(t *testing.T) {
	for _, sync := range []bool{false, true} {
		success, failed := false, false
		for fault := int64(0); fault < 64; fault++ {
			m, budget := rationalManager(t, sync)
			a := Rational{num: NewInt(-2147483648), den: NewInt(-2147483648)}
			out := NewRational()
			qopsMust(t, m.SetInt64(&out, 71))
			var bank []Int
			for m.z.cells.Stats().Live+m.z.cells.Stats().Retired < m.z.cells.Stats().Slots {
				var v Int
				qopsMust(t, m.z.SetUint64(&v, 1<<40))
				bank = append(bank, v)
			}
			roots := []*Int{&a.num, &a.den, &out.num, &out.den}
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
			num := powerPublicationComponent(t, m.z, &out.num, "-2147483648", "2147483648")
			den := powerPublicationComponent(t, m.z, &out.den, "-2147483648", "2147483648")
			if num == "-2147483648" && den != "-2147483648" {
				t.Fatal("denominator Neg preceded numerator Neg")
			}
			if success && (num != "2147483648" || den != "2147483648") {
				t.Fatal("raw reciprocal never normalizes 1/1 fields", out)
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&out))
			for i := range bank {
				qopsMust(t, m.z.Del(&bank[i]))
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
