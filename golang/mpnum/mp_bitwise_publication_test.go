// Licensed under the MIT license.
// Added action-prefix/profile tests for mpz.cpp bitwise and mpq.h forwards.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.
package mpnum

import (
	"errors"
	"math/big"
	"testing"
)

func bitPublicationCall(m *IntegerManager, op int, a, b, out *Int) error {
	switch op {
	case 0:
		return m.BitwiseOr(a, b, out)
	case 1:
		return m.BitwiseAnd(a, b, out)
	case 2:
		return m.BitwiseXor(a, b, out)
	default:
		return m.BitwiseNot(129, a, out)
	}
}

// Allowed publication values are derived independently from source chunk
// accumulation; failed descriptors are checked by the ownership audit first.
func bitPublicationPrefix(t *testing.T, m *IntegerManager, out *Int, old string, a, b *big.Int, op int, width uint) {
	t.Helper()
	if out.cell == m.failedRef {
		return
	}
	got := roundPublicationDecode(t, m, out).String()
	if got == old || got == "0" {
		return
	}
	result := new(big.Int)
	// This helper models positive-input source publication only. Binary loops
	// stop when the shorter operand quotient becomes zero; OR/XOR then publish
	// one full tail, rather than fictitious intervening high chunks.
	if a.Sign() < 0 || b.Sign() < 0 {
		t.Fatal("prefix helper requires positive operands")
	}
	limit := a.BitLen()
	if b.BitLen() < limit {
		limit = b.BitLen()
	}
	if op == 3 {
		limit = int(width)
	}
	for bits := 64; bits < limit+64; bits += 64 {
		end := bits
		if op == 3 && end > limit {
			end = limit
		}
		mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(end)), big.NewInt(1))
		switch op {
		case 0:
			result.Or(a, b)
		case 1:
			result.And(a, b)
		case 2:
			result.Xor(a, b)
		case 3:
			result.Not(a)
		}
		result.And(result, mask)
		if got == result.String() {
			return
		}
	}
	if op == 0 || op == 2 {
		if op == 0 {
			result.Or(a, b)
		} else {
			result.Xor(a, b)
		}
		if got == result.String() {
			return
		}
	}
	t.Fatal("not a completed source chunk/tail prefix", op, got, old)
}

func TestBitwisePublicationNegativeInlineProfile(t *testing.T) {
	for op := 0; op < 4; op++ {
		for alias := 0; alias < 3; alias++ {
			m, _ := mpzPublicationOwner(t)
			a, b, out := NewInt(-3), NewInt(6), NewInt(71)
			dst := []*Int{&out, &a, &b}[alias]
			var e error
			if op == 3 {
				e = m.BitwiseNot(7, &a, dst)
			} else {
				e = bitPublicationCall(m, op, &a, &b, dst)
			}
			mpzPublicationAudit(t, m, &a, &b, &out)
			if modularSourceDebug {
				if !errors.Is(e, ErrBitwiseDomain) {
					t.Fatal(op, e)
				}
				continue
			}
			qopsMust(t, e)
			want := []int64{-1, 4, -5, 2}[op]
			got, err := m.Int64(dst)
			qopsMust(t, err)
			if got != want {
				t.Fatal(op, alias, got, want)
			}
		}
	}
}
func TestBitwisePublicationCallerPrefixes(t *testing.T) {
	for op := 0; op < 4; op++ {
		for alias := 0; alias < 3; alias++ {
			changed, success := false, false
			for fault := int64(0); fault < 256; fault++ {
				m, budget := mpzPublicationOwner(t)
				var a, b, out Int
				qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981"))
				qopsMust(t, m.SetString(&b, "18446744073709551617"))
				qopsMust(t, m.SetInt64(&out, 71))
				dst := []*Int{&out, &a, &b}[alias]
				old := *dst
				before := integerString(t, m, dst)
				aa, _ := new(big.Int).SetString("1461501637330902918203684832716283019655932542981", 10)
				bb, _ := new(big.Int).SetString("18446744073709551617", 10)
				budget.FailAfter(fault)
				e := bitPublicationCall(m, op, &a, &b, dst)
				mpzPublicationAudit(t, m, &a, &b, &out)
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(op, alias, fault, e)
					}
					if *dst != old {
						changed = true
					}
				} else {
					success = true
				}
				budget.FailAfter(-1)
				if e != nil {
					bitPublicationPrefix(t, m, dst, before, aa, bb, op, 129)
				}
				if e != nil && dst.cell != m.failedRef && integerString(t, m, dst) != before {
					changed = true
				}
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
				m.Close()
				if success {
					break
				}
			}
			if (!changed && alias == 0) || !success {
				t.Fatal("source reset/chunk publication before later failure", op, alias, changed, success)
			}
		}
	}
}
func TestBitwisePublicationRationalRawProfile(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			m, _ := rationalManager(t, sync)
			a, b, out := NewRational(), NewRational(), NewRational()
			qopsMust(t, m.z.SetInt64(&a.num, 5))
			qopsMust(t, m.z.SetInt64(&a.den, -2))
			qopsMust(t, m.SetInt64(&b, 3))
			var e error
			switch op {
			case 0:
				e = m.BitwiseOr(&a, &b, &out)
			case 1:
				e = m.BitwiseAnd(&a, &b, &out)
			case 2:
				e = m.BitwiseXor(&a, &b, &out)
			case 3:
				e = m.BitwiseNot(7, &a, &out)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
			if modularSourceDebug {
				if !errors.Is(e, ErrRationalNotInteger) {
					t.Fatal(e)
				}
			} else {
				qopsMust(t, e)
				qopsEqual(t, m, &out, new(big.Rat).SetInt64([]int64{7, 1, 6, 122}[op]))
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&b))
			qopsMust(t, m.Del(&out))
		}
	}
}

func TestBitwisePublicationActualNonterminationBoundary(t *testing.T) {
	for op := 0; op < 3; op++ {
		m, _ := mpzPublicationOwner(t)
		var a, b, out Int
		qopsMust(t, m.SetString(&a, "-18446744073709551617"))
		qopsMust(t, m.SetString(&b, "-18446744073709551618"))
		qopsMust(t, m.SetInt64(&out, 71))
		e := bitPublicationCall(m, op, &a, &b, &out)
		mpzPublicationAudit(t, m, &a, &b, &out)
		if modularSourceDebug {
			if !errors.Is(e, ErrBitwiseDomain) {
				t.Fatal(e)
			}
		} else {
			var stopped *BitwiseNonterminationError
			if !errors.Is(e, ErrNonterminatingBitwise) || !errors.As(e, &stopped) || stopped.Operation != BitwiseOperation(op) {
				t.Fatal(e)
			}
			// Two finite chunks have been published; the next continuation has both
			// quotients -1. No claim is made about future source budget exhaustion.
			aa := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1)))
			bb := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(2)))
			want := new(big.Int)
			switch op {
			case 0:
				want.Or(aa, bb)
			case 1:
				want.And(aa, bb)
			case 2:
				want.Xor(aa, bb)
			}
			assertIntegerPowerOracle(t, m, &out, want)
		}
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
		qopsMust(t, m.Del(&out))
	}
}

func TestBitwisePublicationFiniteMixedSignsAndWidthAliases(t *testing.T) {
	for _, text := range []string{"-3", "-18446744073709551617", "0", "18446744073709551621"} {
		for op := 0; op < 4; op++ {
			for alias := 0; alias < 4; alias++ {
				m, _ := mpzPublicationOwner(t)
				var a, b, out Int
				qopsMust(t, m.SetString(&a, text))
				qopsMust(t, m.SetString(&b, "18446744073709551621"))
				out = NewInt(71)
				left, right, dst := &a, &b, &out
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
				aa, _ := new(big.Int).SetString(text, 10)
				bb := new(big.Int).SetUint64(0)
				bb.SetString("18446744073709551621", 10)
				if alias == 3 {
					bb.Set(aa)
				}
				// Both-negative binary general paths belong to the separate boundary test.
				if op != 3 && aa.Sign() < 0 && bb.Sign() < 0 {
					continue
				}
				e := bitPublicationCall(m, op, left, right, dst)
				mpzPublicationAudit(t, m, &a, &b, &out)
				if modularSourceDebug && (aa.Sign() < 0 || op != 3 && bb.Sign() < 0) {
					if !errors.Is(e, ErrBitwiseDomain) {
						t.Fatal(e)
					}
				} else {
					qopsMust(t, e)
					want := new(big.Int)
					switch op {
					case 0:
						want.Or(aa, bb)
					case 1:
						want.And(aa, bb)
					case 2:
						want.Xor(aa, bb)
					case 3:
						want.And(new(big.Int).Not(aa), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 129), big.NewInt(1)))
					}
					assertIntegerPowerOracle(t, m, dst, want)
				}
				qopsMust(t, m.Del(&a))
				qopsMust(t, m.Del(&b))
				qopsMust(t, m.Del(&out))
			}
		}
	}
}

func TestBitwisePublicationFinalUnusedActions(t *testing.T) {
	for op := 0; op < 4; op++ {
		completeFailure, success := false, false
		for fault := int64(0); fault < 256; fault++ {
			m, budget := mpzPublicationDirectOwner(t)
			var a, b, out Int
			qopsMust(t, m.SetUint64(&a, ^uint64(0)))
			qopsMust(t, m.SetUint64(&b, ^uint64(0)-1))
			out = NewInt(71)
			budget.FailAfter(fault)
			var e error
			if op == 3 {
				e = m.BitwiseNot(64, &a, &out)
			} else {
				e = bitPublicationCall(m, op, &a, &b, &out)
			}
			mpzPublicationAudit(t, m, &a, &b, &out)
			budget.FailAfter(-1)
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				if out.cell != m.failedRef {
					want := []string{"18446744073709551615", "18446744073709551614", "1", "0"}[op]
					if integerString(t, m, &out) == want {
						completeFailure = true
					}
				}
			} else {
				success = true
			}
			qopsMust(t, m.Del(&a))
			qopsMust(t, m.Del(&b))
			qopsMust(t, m.Del(&out))
			m.Close()
			if success {
				break
			}
		}
		if !completeFailure || !success {
			t.Fatal("completed result precedes final unused multiplier/division", op, completeFailure, success)
		}
	}
}

func TestBitwisePublicationRationalDenominatorFailureAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			for alias := 0; alias < 3; alias++ {
				changed, success := false, false
				for fault := int64(0); fault < 256; fault++ {
					m, budget := rationalPublicationDirectOwner(t, sync)
					a, b, out := NewRational(), NewRational(), NewRational()
					qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981"))
					qopsMust(t, m.SetString(&b, "18446744073709551617"))
					qopsMust(t, m.SetFraction(&out, 3, 7))
					dst := []*Rational{&out, &a, &b}[alias]
					den := dst.den
					before := integerString(t, m.z, &dst.num)
					budget.FailAfter(fault)
					var e error
					switch op {
					case 0:
						e = m.BitwiseOr(&a, &b, dst)
					case 1:
						e = m.BitwiseAnd(&a, &b, dst)
					case 2:
						e = m.BitwiseXor(&a, &b, dst)
					case 3:
						e = m.BitwiseNot(129, &a, dst)
					}
					qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &b.num, &b.den, &out.num, &out.den})
					if e != nil {
						if !numericPhysicalReservationRefusal(e) {
							t.Fatal(op, alias, fault, e)
						}
						if dst.den != den {
							t.Fatal("den reset before integer success", op, alias, fault)
						}
					} else {
						success = true
						if !isOneInteger(&dst.den) {
							t.Fatal("missing den reset")
						}
					}
					budget.FailAfter(-1)
					if e != nil {
						aa, _ := new(big.Int).SetString("1461501637330902918203684832716283019655932542981", 10)
						bb, _ := new(big.Int).SetString("18446744073709551617", 10)
						bitPublicationPrefix(t, m.z, &dst.num, before, aa, bb, op, 129)
					}
					if e != nil && (dst.num.cell == m.z.failedRef || integerString(t, m.z, &dst.num) != before) {
						changed = true
					}
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&b))
					qopsMust(t, m.Del(&out))
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("cleanup", budget.Used())
					}
					if success {
						break
					}
				}
				if (!changed && alias == 0) || !success {
					t.Fatal(op, alias, changed, success)
				}
			}
		}
	}
}

func TestBitwisePublicationEarlyFailurePrecedesBoundary(t *testing.T) {
	if modularSourceDebug {
		return
	} // source assertion precedes copies in debug
	for op := 0; op < 3; op++ {
		m, budget := mpzPublicationDirectOwner(t)
		var a, b, out Int
		qopsMust(t, m.SetString(&a, "-18446744073709551617"))
		qopsMust(t, m.SetString(&b, "-18446744073709551618"))
		out = NewInt(71)
		budget.FailAfter(0)
		e := bitPublicationCall(m, op, &a, &b, &out)
		mpzPublicationAudit(t, m, &a, &b, &out)
		if !numericPhysicalReservationRefusal(e) || out != NewInt(71) {
			t.Fatal("copy refusal before output reset/boundary", op, e, out)
		}
		budget.FailAfter(-1)
		qopsMust(t, m.Del(&a))
		qopsMust(t, m.Del(&b))
	}
}

func TestBitwisePublicationServiceValueRoutes(t *testing.T) {
	h := facadeNew(t)
	a := h.r("18446744073709551621")
	b := h.r("18446744073709551617")
	for op := 0; op < 4; op++ {
		var out OwnedRational
		var e error
		switch op {
		case 0:
			out, e = h.s.BitwiseOr(a, b)
		case 1:
			out, e = h.s.BitwiseAnd(a, b)
		case 2:
			out, e = h.s.BitwiseXor(a, b)
		case 3:
			out, e = h.s.BitwiseNot(65, a)
		}
		serviceMust(t, e)
		held := h.keep(out, nil)
		facadeAudit(t, h.s, h.owned...)
		text, e := h.s.String(held)
		serviceMust(t, e)
		want := []string{"18446744073709551621", "18446744073709551617", "4", "18446744073709551610"}[op]
		if text != want {
			t.Fatal(op, text, want)
		}
	}
}

func TestBitwisePublicationServiceFailureRoots(t *testing.T) {
	for op := 0; op < 4; op++ {
		failed, success := false, false
		for fault := int64(0); fault < 512; fault++ {
			s, budget := servicePublicationDirectOwner(t)
			a := serviceValue(t, s, "1461501637330902918203684832716283019655932542981")
			b := serviceValue(t, s, "18446744073709551617")
			beforeA, e := s.String(&a)
			serviceMust(t, e)
			beforeB, e := s.String(&b)
			serviceMust(t, e)
			budget.FailAfter(fault)
			var out OwnedRational
			switch op {
			case 0:
				out, e = s.BitwiseOr(&a, &b)
			case 1:
				out, e = s.BitwiseAnd(&a, &b)
			case 2:
				out, e = s.BitwiseXor(&a, &b)
			case 3:
				out, e = s.BitwiseNot(129, &a)
			}
			facadeAudit(t, s, &a, &b, &out)
			if e != nil {
				failed = true
				if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
					t.Fatal(op, fault, e, out)
				}
			} else {
				success = true
			}
			budget.FailAfter(-1)
			afterA, e := s.String(&a)
			serviceMust(t, e)
			afterB, e := s.String(&b)
			serviceMust(t, e)
			if beforeA != afterA || beforeB != afterB {
				t.Fatal("value wrapper mutated inputs", op, fault)
			}
			if out.ownerID != 0 {
				serviceMust(t, s.Del(&out))
			}
			serviceMust(t, s.Del(&a))
			serviceMust(t, s.Del(&b))
			serviceMust(t, s.Close())
			if budget.Used() != 0 {
				t.Fatal("cold graph leaked", budget.Used())
			}
			if success {
				break
			}
		}
		if !failed || !success {
			t.Fatal(op, failed, success)
		}
	}
}

func TestBitwisePublicationBigNotZeroWidthCopyOrder(t *testing.T) {
	m, budget := mpzPublicationDirectOwner(t)
	var a, out Int
	qopsMust(t, m.SetString(&a, "18446744073709551617"))
	out = NewInt(71)
	var bank []Int
	for m.cells.Stats().Live+m.cells.Stats().Retired < m.cells.Stats().Slots {
		var v Int
		qopsMust(t, m.Copy(&v, &a))
		bank = append(bank, v)
	}
	roots := []*Int{&a, &out}
	for i := range bank {
		roots = append(roots, &bank[i])
	}
	budget.FailAfter(0)
	e := m.BitwiseNot(0, &a, &out)
	mpzPublicationAudit(t, m, roots...)
	if !numericPhysicalReservationRefusal(e) || out != NewInt(71) {
		t.Fatal("source copy precedes zero-width output reset", e, out)
	}
	budget.FailAfter(-1)
	qopsMust(t, m.BitwiseNot(0, &a, &out))
	mpzPublicationAudit(t, m, roots...)
	if out != NewInt(0) {
		t.Fatal(out)
	}
	for i := range bank {
		qopsMust(t, m.Del(&bank[i]))
	}
	qopsMust(t, m.Del(&a))
}

func TestBitwisePublicationSignedSmallBoundaryAliases(t *testing.T) {
	for _, av := range []int32{-2147483648, -1, -3} {
		for _, bv := range []int32{-2147483648, -1, 6} {
			for op := 0; op < 3; op++ {
				for alias := 0; alias < 4; alias++ {
					m, _ := mpzPublicationOwner(t)
					a, b, out := NewInt(av), NewInt(bv), NewInt(71)
					left, right, dst := &a, &b, &out
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
					e := bitPublicationCall(m, op, left, right, dst)
					mpzPublicationAudit(t, m, &a, &b, &out)
					if modularSourceDebug {
						if !errors.Is(e, ErrBitwiseDomain) {
							t.Fatal(e)
						}
					} else {
						qopsMust(t, e)
						rhs := bv
						if alias == 3 {
							rhs = av
						}
						want := int32(0)
						switch op {
						case 0:
							want = av | rhs
						case 1:
							want = av & rhs
						case 2:
							want = av ^ rhs
						}
						if *dst != NewInt(want) {
							t.Fatal(av, bv, op, alias, *dst, want)
						}
					}
				}
			}
		}
	}
}
