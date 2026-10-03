// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Source: src/util/mpq.cpp floor/ceil and mpq.h output wrappers.
package mpnum

import (
	"fmt"
	"math"
	"math/big"
	"testing"

)

func roundPublicationFillCells(t *testing.T, m *IntegerManager) []*Int {
	t.Helper()
	var bank []*Int
	for m.cells.Stats().Live+m.cells.Stats().Retired < m.cells.Stats().Slots {
		var v Int
		qopsMust(t, m.SetUint64(&v, 4294967297))
		bank = append(bank, &v)
	}
	return bank
}
func roundPublicationCall(m *RationalManager, a, out *Rational, integer *Int, op int) error {
	switch op {
	case 0:
		return m.FloorToInt(a, integer)
	case 1:
		return m.CeilToInt(a, integer)
	case 2:
		return m.Floor(a, out)
	default:
		return m.Ceil(a, out)
	}
}
func TestRoundPublicationIntegralCopyRefusal(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			for _, oldBig := range []bool{false, true} {
				for _, negative := range []bool{false, true} {
					q, budget := rationalManager(t, sync)
					a, out := NewRational(), NewRational()
					integer := NewInt(17)
					limbs := make([]uint32, 65)
					for i := range limbs {
						limbs[i] = uint32(17*i + 1)
					}
					qopsMust(t, q.z.SetLimbs(&a.num, limbs))
					if negative {
						qopsMust(t, q.z.Neg(&a.num))
					}
					out.num = NewInt(17)
					qopsMust(t, q.z.SetUint64(&out.den, 4294967297))
					dst := &integer
					if op >= 2 {
						dst = &out.num
					}
					if oldBig {
						qopsMust(t, q.z.SetUint64(dst, 4294967297))
					}
					old := dst.cell
					beforeA, beforeDen := a, out.den
					budget.FailAfter(0)
					e := roundPublicationCall(q, &a, &out, &integer, op)
					qopsAuditLiveOwnership(t, q, []*Int{&a.num, &a.den, &out.num, &out.den, &integer})
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(sync, op, oldBig, negative, e)
					}
					if dst.val != a.num.val || dst.cell != q.z.failedRef {
						t.Errorf("actual caller Copy prefix hidden: sync=%v op=%d oldBig=%v negative=%v dst=%+v", sync, op, oldBig, negative, *dst)
					}
					if old != 0 && validNumericRef(q.z.cells, old) {
						t.Error("source old caller BIG was not freed before Copy refusal")
					}
					if a != beforeA || out.den != beforeDen {
						t.Fatal("source inputs/uncalled denominator reset changed")
					}
					budget.FailAfter(-1)
					for _, v := range []*Int{&integer, &out.den, &out.num, &a.den, &a.num} {
						qopsMust(t, q.z.Del(v))
					}
					q.Close()
					if budget.Used() != 0 {
						t.Fatal("Copy refusal leaked", budget.Used())
					}
				}
			}
		}
	}
}
func TestRoundPublicationQuotientBeforeCeilCorrectionRefusal(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, rationalOut := range []bool{false, true} {
			q, budget := rationalManager(t, sync)
			a, out := NewRational(), NewRational()
			integer := NewInt(17)
			// This deliberately unreduced source shape has an exact MAX_INT quotient;
			// source Ceil still adds1 because it tests numerator sign, not remainder.
			qopsMust(t, q.z.SetUint64(&a.num, 4294967294))
			a.den = NewInt(2)
			out.num = NewInt(17)
			out.den = NewInt(5)
			// Warm only the actual integer division scratch; do not call Ceil or repair
			// any faulted source cache. Its inline quotient publication cannot allocate.
			var warm Int
			qopsMust(t, q.z.MachineDiv(&a.num, &a.den, &warm))
			qopsMust(t, q.z.Del(&warm))
			bank := roundPublicationFillCells(t, q.z)
			roots := append([]*Int{&a.num, &a.den, &out.num, &out.den, &integer}, bank...)
			dst := &integer
			op := 1
			if rationalOut {
				dst = &out.num
				op = 3
			}
			beforeA, beforeDen := a, out.den
			budget.FailAfter(0)
			e := roundPublicationCall(q, &a, &out, &integer, op)
			qopsAuditLiveOwnership(t, q, roots)
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(sync, rationalOut, e)
			}
			if *dst != NewInt(math.MaxInt32) {
				t.Errorf("actual quotient before correction refusal hidden: sync=%v rational=%v dst=%+v", sync, rationalOut, *dst)
			}
			if a != beforeA || out.den != beforeDen {
				t.Fatal("source input/denominator prefix changed")
			}
			budget.FailAfter(-1)
			for i := len(roots) - 1; i >= 0; i-- {
				qopsMust(t, q.z.Del(roots[i]))
			}
			q.Close()
			if budget.Used() != 0 {
				t.Fatal("correction refusal leaked", budget.Used())
			}
		}
	}
}

func TestRoundPublicationRawShapesAndAllComponentAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		q, _ := rationalManager(t, sync)
		for _, n := range []int32{math.MinInt32, -6, -2, -1, 0, 1, 2, 6, math.MaxInt32} {
			for _, d := range []int32{-3, -2, -1, 1, 2, 3} {
				for _, ceil := range []bool{false, true} {
					// Literal source sign-only correction is deliberately different from a
					// mathematical oracle on negative or unreduced denominators.
					want := int64(n)
					if d != 1 {
						want = int64(n) / int64(d)
						if ceil && n > 0 {
							want++
						}
						if !ceil && n < 0 {
							want--
						}
					}
					for alias := 0; alias < 5; alias++ {
						a, out := NewRational(), NewRational()
						integer := NewInt(17)
						a.num, a.den = NewInt(n), NewInt(d)
						out.num, out.den = NewInt(17), NewInt(5)
						beforeA := a
						var e error
						var result *Int
						if alias < 3 {
							result = []*Int{&integer, &a.num, &a.den}[alias]
							if ceil {
								e = q.CeilToInt(&a, result)
							} else {
								e = q.FloorToInt(&a, result)
							}
						} else {
							dst := &out
							if alias == 4 {
								dst = &a
							}
							result = &dst.num
							if ceil {
								e = q.Ceil(&a, dst)
							} else {
								e = q.Floor(&a, dst)
							}
							if e == nil && dst.den != NewInt(1) {
								t.Fatal("source final denominator reset", sync, n, d, ceil, alias)
							}
						}
						qopsAuditLiveOwnership(t, q, []*Int{&a.num, &a.den, &out.num, &out.den, &integer})
						if e != nil || integerString(t, q.z, result) != fmt.Sprint(want) {
							t.Fatal("literal source raw rounding", sync, n, d, ceil, alias, e, want)
						}
						if (alias == 0 || alias == 3) && a != beforeA {
							t.Fatal("nonoutput source changed")
						}
						for _, v := range []*Int{&integer, &out.den, &out.num, &a.den, &a.num} {
							qopsMust(t, q.z.Del(v))
						}
					}
				}
			}
		}
	}
}

// Decode actual scalar/limbs without invoking the formatter or allocating any
// numerical owner storage. The independent oracle is test-only.
func roundPublicationDecode(t *testing.T, m *IntegerManager, n *Int) *big.Int {
	t.Helper()
	qopsMust(t, m.check(n))
	if n.IsSmall() {
		return big.NewInt(int64(n.val))
	}
	value := new(big.Int)
	cell := m.cell(n.cell)
	for i := cell.size; i > 0; i-- {
		value.Lsh(value, 32)
		value.Add(value, new(big.Int).SetUint64(uint64(m.digit(n.cell, i-1))))
	}
	if n.val < 0 {
		value.Neg(value)
	}
	return value
}
