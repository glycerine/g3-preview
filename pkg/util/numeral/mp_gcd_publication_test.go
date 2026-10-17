// Added source-action/profile tests for selected internal32 GCD methods.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func gcdPublicationFillCells(t *testing.T, m *IntegerManager) []*Int {
	t.Helper()
	var bank []*Int
	for m.Cells.Stats().Live+m.Cells.Stats().Retired < m.Cells.Stats().Slots {
		var v Int
		qopsMust(t, m.SetUint64(&v, 4294967297))
		bank = append(bank, &v)
	}
	return bank
}
func TestGCDPublicationZeroCopyBeforeMinimumPromotion(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			for alias := 0; alias < 3; alias++ {
				q, budget := rationalPublicationDirectOwner(t, sync)
				m := q.Z
				zero, minimum, out := NewInt(0), NewInt(math.MinInt32), NewInt(17)
				dst := []*Int{&out, &zero, &minimum}[alias]
				bank := gcdPublicationFillCells(t, m)
				a, b := &zero, &minimum
				if reverse {
					a, b = b, a
				}
				budget.FailAfter(0)
				e := m.GCD(a, b, dst)
				qopsAuditLiveOwnership(t, q, append([]*Int{&zero, &minimum, &out}, bank...))
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(sync, reverse, alias, e)
				}
				// Source Set(c,INT_MIN) precedes Abs(c)'s actual promotion allocation.
				// Allocation fails before its pointer/new positive sign assignment; the
				// completed Copy remains a valid inline INT_MIN numeral.
				if dst.Val != math.MinInt32 || dst.Cell != 0 {
					t.Errorf("source Copy before Abs refusal hidden: synch=%v reverse=%v alias=%d out=%+v", sync, reverse, alias, *dst)
				}
				if dst != &zero && zero != NewInt(0) {
					t.Fatal("non-destination zero changed")
				}
				if dst != &minimum && minimum != NewInt(math.MinInt32) {
					t.Fatal("non-destination minimum changed")
				}
				budget.FailAfter(-1)
				for _, v := range append([]*Int{&zero, &minimum, &out}, bank...) {
					qopsMust(t, m.Del(v))
				}
				q.Close()
				if budget.Used() != 0 {
					t.Fatal("GCD graph leak", budget.Used())
				}
			}
		}
	}
}
func TestGCDPublicationZeroBigCopyRefusal(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			q, budget := rationalManager(t, sync)
			m := q.Z
			zero, other, out := NewInt(0), NewInt(0), NewInt(0)
			limbs := make([]uint32, 65)
			for i := range limbs {
				limbs[i] = uint32(i*17 + 1)
			}
			qopsMust(t, m.SetLimbs(&other, limbs))
			qopsMust(t, m.Neg(&other))
			qopsMust(t, m.SetUint64(&out, 4294967297))
			old := out.Cell
			a, b := &zero, &other
			if reverse {
				a, b = b, a
			}
			budget.FailAfter(0)
			e := m.GCD(a, b, &out)
			qopsAuditLiveOwnership(t, q, []*Int{&zero, &other, &out})
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(sync, reverse, e)
			}
			if out.Val != -1 || out.Cell != m.FailedRef || ValidNumericRef(m.Cells, old) {
				t.Errorf("source destination sign/free/Copy refusal hidden: %v %v %+v oldLive=%v", sync, reverse, out, ValidNumericRef(m.Cells, old))
			}
			if zero != NewInt(0) || other.Val != -1 {
				t.Fatal("untouched source inputs")
			}
			budget.FailAfter(-1)
			for _, v := range []*Int{&zero, &other, &out} {
				qopsMust(t, m.Del(v))
			}
			q.Close()
			if budget.Used() != 0 {
				t.Fatal("copy refusal graph leak", budget.Used())
			}
		}
	}
}
func TestGCDPublicationRationalIntegralityProfiles(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "TTTTTTTT")
	}
	for _, sync := range []bool{false, true} {
		for operation := 0; operation < 4; operation++ {
			m, _ := rationalManager(t, sync)
			r1, r2, a, b, g := NewRational(), NewRational(), NewRational(), NewRational(), NewRational()
			r1.Num, r1.Den = NewInt(6), NewInt(-5)
			r2.Num, r2.Den = NewInt(10), NewInt(7)
			qopsMust(t, m.SetFraction(&a, 17, 3))
			qopsMust(t, m.SetFraction(&b, 19, 5))
			qopsMust(t, m.SetFraction(&g, 23, 7))
			beforeA, beforeB, beforeG := a, b, g
			var e error
			var divides bool
			switch operation {
			case 0:
				e = m.GCD(&r1, &r2, &g)
			case 1:
				e = m.LCM(&r1, &r2, &g)
			case 2:
				r1.Num = NewInt(2)
				r2.Num = NewInt(6)
				divides, e = m.Divides(&r1, &r2)
			case 3:
				e = m.ExtendedGCD(&r1, &r2, &a, &b, &g)
			}
			qopsAuditLiveOwnership(t, m, []*Int{&r1.Num, &r1.Den, &r2.Num, &r2.Den, &a.Num, &a.Den, &b.Num, &b.Den, &g.Num, &g.Den})
			if modularSourceDebug {
				if !isMPNVerificationThrow(e) || a != beforeA || b != beforeB || g != beforeG {
					t.Fatal("source integrality assertion before actions", operation, e, a, b, g)
				}
			} else {
				if e != nil {
					t.Errorf("finite release raw numerator action rejected: synch=%v op=%d err=%v", sync, operation, e)
				} else {
					switch operation {
					case 0:
						if g != (Rational{Num: NewInt(2), Den: NewInt(1)}) {
							t.Fatal(g)
						}
					case 1:
						if g != (Rational{Num: NewInt(30), Den: NewInt(1)}) {
							t.Fatal(g)
						}
					case 2:
						if !divides || a != beforeA || b != beforeB || g != beforeG {
							t.Fatal("raw numerator divisibility", divides)
						}
					case 3:
						if a != (Rational{Num: NewInt(2), Den: NewInt(1)}) || b != (Rational{Num: NewInt(-1), Den: NewInt(1)}) || g != (Rational{Num: NewInt(2), Den: NewInt(1)}) {
							t.Fatal("source coefficient and denominator prefixes", a, b, g)
						}
					}
				}
			}
			for _, v := range []*Rational{&r1, &r2, &a, &b, &g} {
				qopsMust(t, m.Del(v))
			}
		}
	}
}
func TestGCDPublicationRationalArrayVisitedFraction(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "TT")
	}
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		as := []Rational{{Num: NewInt(6), Den: NewInt(1)}, {Num: NewInt(10), Den: NewInt(1)}, {Num: NewInt(3), Den: NewInt(5)}}
		g := NewRational()
		e := m.GCDMany(as, &g)
		qopsAuditLiveOwnership(t, m, []*Int{&as[0].Num, &as[0].Den, &as[1].Num, &as[1].Den, &as[2].Num, &as[2].Den, &g.Num, &g.Den})
		if modularSourceDebug {
			if !isMPNVerificationThrow(e) || g.Num != NewInt(2) || g.Den != NewInt(1) {
				t.Fatal("actual visited assertion prefix", e, g)
			}
		} else {
			if e != nil || g.Num != NewInt(1) || g.Den != NewInt(1) {
				t.Errorf("source finite visited fractional numerator: %v %+v", e, g)
			}
		}
		for i := range as {
			qopsMust(t, m.Del(&as[i]))
		}
		qopsMust(t, m.Del(&g))
	}
}

// Deterministic hermetic replacement for libc rand: nonnegative31-bit values,
// then exactly source %ratio, optional %max_pw+1, and final %2 draw order.
// It explicitly does not reproduce a libc sequence or global mutable state.
type gcdPublicationRNG struct {
	state uint32
	draws uint32
}

func (r *gcdPublicationRNG) next() uint32 {
	r.state = r.state*1664525 + 1013904223
	r.draws++
	return r.state >> 1
}

var gcdPublicationPrimes168 = [168]int32{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199, 211, 223, 227, 229, 233, 239, 241, 251, 257, 263, 269, 271, 277, 281, 283, 293, 307, 311, 313, 317, 331, 337, 347, 349, 353, 359, 367, 373, 379, 383, 389, 397, 401, 409, 419, 421, 431, 433, 439, 443, 449, 457, 461, 463, 467, 479, 487, 491, 499, 503, 509, 521, 523, 541, 547, 557, 563, 569, 571, 577, 587, 593, 599, 601, 607, 613, 617, 619, 631, 641, 643, 647, 653, 659, 661, 673, 677, 683, 691, 701, 709, 719, 727, 733, 739, 743, 751, 757, 761, 769, 773, 787, 797, 809, 811, 821, 823, 827, 829, 839, 853, 857, 859, 863, 877, 881, 883, 887, 907, 911, 919, 929, 937, 941, 947, 953, 967, 971, 977, 983, 991, 997}

func gcdPublicationSourceGenerate(t *testing.T, m *IntegerManager, rng *gcdPublicationRNG, out *Int) *big.Int {
	t.Helper()
	var tmp Int
	defer m.Del(&tmp)
	qopsMust(t, m.SetInt32(out, 1))
	oracle := big.NewInt(1)
	for _, prime := range gcdPublicationPrimes168 {
		if rng.next()%3 == 0 {
			power := rng.next()%2 + 1
			base := NewInt(prime)
			qopsMust(t, m.Power(&base, power, &tmp))
			qopsMust(t, m.Mul(out, &tmp, out))
			oracle.Mul(oracle, new(big.Int).Exp(big.NewInt(int64(prime)), new(big.Int).SetUint64(uint64(power)), nil))
		}
	}
	if rng.next()%2 == 0 {
		qopsMust(t, m.Neg(out))
		oracle.Neg(oracle)
	}
	return oracle
}
func gcdPublicationSourceSlow(t *testing.T, m *IntegerManager, a, b, out *Int) {
	t.Helper()
	var one, two, aux Int
	defer func() { m.Del(&aux); m.Del(&two); m.Del(&one) }()
	qopsMust(t, m.Copy(&one, a))
	qopsMust(t, m.Copy(&two, b))
	qopsMust(t, m.Abs(&one))
	qopsMust(t, m.Abs(&two))
	cmp, e := m.Compare(&one, &two)
	qopsMust(t, e)
	if cmp <= 0 {
		qopsMust(t, m.Swap(&one, &two))
	}
	if two.Val == 0 {
		qopsMust(t, m.Copy(out, &one))
		return
	}
	for {
		qopsMust(t, m.Rem(&one, &two, &aux))
		if aux.Val == 0 {
			qopsMust(t, m.Copy(out, &two))
			return
		}
		qopsMust(t, m.Copy(&one, &two))
		qopsMust(t, m.Copy(&two, &aux))
	}
}
func TestGCDPublicationSourceFull168PrimeCampaign(t *testing.T) {
	q, _ := rationalManager(t, false)
	m := q.Z
	rng := gcdPublicationRNG{state: 0x47434431}
	var a, b, g1, g2 Int
	for i := 0; i < 50; i++ {
		ao := gcdPublicationSourceGenerate(t, m, &rng, &a)
		bo := gcdPublicationSourceGenerate(t, m, &rng, &b)
		gcdPublicationSourceSlow(t, m, &a, &b, &g1)
		qopsMust(t, m.GCD(&a, &b, &g2))
		qopsAuditLiveOwnership(t, q, []*Int{&a, &b, &g1, &g2})
		cmp, e := m.Compare(&g1, &g2)
		qopsMust(t, e)
		if cmp != 0 {
			t.Fatal("source conditional UNREACHABLE", i)
		}
		want := new(big.Int).GCD(nil, nil, ao, bo)
		if integerString(t, m, &a) != ao.String() || integerString(t, m, &b) != bo.String() || integerString(t, m, &g2) != want.String() {
			t.Fatal("independent campaign oracle", i)
		}
	}
	if rng.draws < 50*2*169 || rng.draws > 50*2*337 {
		t.Fatal("source conditional draw bounds", rng.draws)
	}
	// Registered tst_gcd_bug is print-only: this is an added exact output check.
	qopsMust(t, m.SetInt32(&a, math.MinInt32))
	qopsMust(t, m.SetInt32(&b, math.MinInt32))
	qopsMust(t, m.GCD(&a, &b, &g2))
	qopsAuditLiveOwnership(t, q, []*Int{&a, &b, &g1, &g2})
	if integerString(t, m, &g2) != "2147483648" {
		t.Fatal("source print expectation")
	}
	for _, v := range []*Int{&g2, &g1, &b, &a} {
		qopsMust(t, m.Del(v))
	}
}

func TestGCDPublicationPrimeTableAndPredicateBoundary(t *testing.T) {
	// Independently enumerate the complete mathematical prime table, rather than
	// letting a mistyped source table feed both the generator and its oracle.
	pos := 0
	for n := int32(2); n <= 997; n++ {
		prime := true
		for d := int32(2); d*d <= n; d++ {
			if n%d == 0 {
				prime = false
				break
			}
		}
		if prime {
			if pos >= len(gcdPublicationPrimes168) || gcdPublicationPrimes168[pos] != n {
				t.Fatal("source full prime table", pos, n)
			}
			pos++
		}
	}
	if pos != 168 {
		t.Fatal(pos)
	}
	q, _ := rationalManager(t, false)
	m := q.Z
	one, two := NewInt(1), NewInt(2)
	e := m.gcdSourceGreaterEqual(&one, &two)
	qopsAuditLiveOwnership(t, q, []*Int{&one, &two})
	if sourceGCDDebug {
		if !errors.Is(e, ErrIntegerAssertion) {
			t.Fatal("actual false ge diagnostic", e)
		}
	} else if e != nil {
		t.Fatal("source release does not evaluate assertion", e)
	}
	qopsMust(t, m.gcdSourceGreaterEqual(&two, &one))
	qopsMust(t, m.gcdSourceGreaterEqual(&one, &one))
	// This checks the actual source predicate helper boundary, not a claim that
	// canonical public GCD violates its proven loop invariant.
}
func TestGCDPublicationFullFamilySignedAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		q, _ := rationalManager(t, sync)
		m := q.Z
		for _, x := range []int32{-15, -6, -1, 0, 1, 6, 15} {
			for _, y := range []int32{-10, -1, 0, 1, 10} {
				for alias := 0; alias < 3; alias++ {
					a, b, out := NewInt(x), NewInt(y), NewInt(71)
					dst := []*Int{&out, &a, &b}[alias]
					qopsMust(t, m.GCD(&a, &b, dst))
					qopsAuditLiveOwnership(t, q, []*Int{&a, &b, &out})
					want := new(big.Int).GCD(nil, nil, big.NewInt(int64(x)), big.NewInt(int64(y)))
					if integerString(t, m, dst) != want.String() {
						t.Fatal("GCD oracle", x, y, alias)
					}
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&b))
					qopsMust(t, m.Del(&out))
					a, b, out = NewInt(x), NewInt(y), NewInt(71)
					qopsMust(t, m.LCM(&a, &b, dst))
					qopsAuditLiveOwnership(t, q, []*Int{&a, &b, &out})
					var lcm *big.Int
					// These literal source shortcuts preserve signed/noncanonical LCM results.
					switch {
					case y == 1:
						lcm = big.NewInt(int64(x))
					case x == 1 || x == y:
						lcm = big.NewInt(int64(y))
					case want.Cmp(big.NewInt(int64(x))) == 0:
						lcm = big.NewInt(int64(y))
					case want.Cmp(big.NewInt(int64(y))) == 0:
						lcm = big.NewInt(int64(x))
					default:
						lcm = new(big.Int).Mul(new(big.Int).Quo(big.NewInt(int64(x)), want), big.NewInt(int64(y)))
					}
					if integerString(t, m, dst) != lcm.String() {
						t.Fatal("source signed LCM", x, y, alias)
					}
					for _, v := range []*Int{&a, &b, &out} {
						qopsMust(t, m.Del(v))
					}
				}
			}
		}
		// Array alias to a future element changes what the later source call sees.
		values := []Int{NewInt(6), NewInt(10), NewInt(9)}
		qopsMust(t, m.GCDMany(values, &values[2]))
		qopsAuditLiveOwnership(t, q, []*Int{&values[0], &values[1], &values[2]})
		if values[2] != NewInt(2) {
			t.Fatal("source future-element array alias", values[2])
		}
	}
}
