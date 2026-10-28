package numeral

// Added action-order tests for the additive and mixed-product mpq.h family.
// Source: src/util/mpz.h, src/util/mpz.cpp, src/util/mpq.h.

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// The dispatcher calls public methods, including the actual mixed product
// prerequisites; it does not implement a second production arithmetic path.
func addPublicationCall(m *RationalManager, op int, a, b, c, out *Rational) error {
	switch op {
	case 0:
		return m.Add(a, b, out)
	case 1:
		return m.Sub(a, b, out)
	case 2:
		return m.AddInt(a, &b.Num, out)
	case 3:
		return m.MulInt(&b.Num, a, out)
	case 4:
		return m.MulInts(&a.Num, &b.Num, out)
	case 5:
		return m.AddMul(a, b, c, out)
	case 6:
		return m.SubMul(a, b, c, out)
	case 7:
		return m.AddMulInt(a, &b.Num, c, out)
	default:
		return m.SubMulInt(a, &b.Num, c, out)
	}
}
func addPublicationModel(op int, a, b, c *big.Rat) *big.Rat {
	r := new(big.Rat)
	switch op {
	case 0:
		return r.Add(a, b)
	case 1:
		return r.Sub(a, b)
	case 2:
		return r.Add(a, new(big.Rat).SetInt(b.Num()))
	case 3:
		return r.Mul(a, new(big.Rat).SetInt(b.Num()))
	case 4:
		return r.SetInt(new(big.Int).Mul(a.Num(), b.Num()))
	case 5:
		return r.Add(a, new(big.Rat).Mul(b, c))
	case 6:
		return r.Sub(a, new(big.Rat).Mul(b, c))
	case 7:
		return r.Add(a, new(big.Rat).Mul(new(big.Rat).SetInt(b.Num()), c))
	default:
		return r.Sub(a, new(big.Rat).Mul(new(big.Rat).SetInt(b.Num()), c))
	}
}
func addPublicationAudit(t *testing.T, m *RationalManager, values ...*Rational) {
	t.Helper()
	roots := make([]*Int, 0, 2*len(values))
	for _, v := range values {
		roots = append(roots, &v.Num, &v.Den)
	}
	qopsAuditLiveOwnership(t, m, roots)
}
func TestAdditivePublicationBoundedModelsAndAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for op := 0; op < 9; op++ {
			for alias := 0; alias < 4; alias++ {
				for n := int64(-3); n <= 3; n++ {
					a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
					qopsMust(t, m.SetString(&a, new(big.Rat).SetFrac64(n, 5).RatString()))
					qopsMust(t, m.SetString(&b, "7/3"))
					qopsMust(t, m.SetString(&c, "-2/11"))
					qopsMust(t, m.SetInt64(&out, 71))
					ar := new(big.Rat).SetFrac64(n, 5)
					br := new(big.Rat).SetFrac64(7, 3)
					cr := new(big.Rat).SetFrac64(-2, 11)
					dst := []*Rational{&out, &a, &b, &c}[alias]
					want := addPublicationModel(op, ar, br, cr)
					qopsMust(t, addPublicationCall(m, op, &a, &b, &c, dst))
					addPublicationAudit(t, m, &a, &b, &c, &out)
					got, e := m.String(dst)
					qopsMust(t, e)
					if got != want.RatString() {
						t.Fatal(sync, op, alias, n, got, want)
					}
					for _, v := range []*Rational{&a, &b, &c, &out} {
						qopsMust(t, m.Del(v))
					}
				}
			}
		}
	}
}

// Positive, canonical input contract: these are precisely the full component
// values at source call boundaries, rather than a blanket permission to mutate.
func addPublicationComponents(op int, a, b, c *big.Rat) (*big.Int, *big.Int) {
	an, ad, bn, bd := a.Num(), a.Denom(), b.Num(), b.Denom()
	switch op {
	case 2:
		return new(big.Int).Add(an, new(big.Int).Mul(bn, ad)), new(big.Int).Set(ad)
	case 3:
		return new(big.Int).Mul(bn, an), new(big.Int).Set(ad)
	case 4:
		return new(big.Int).Mul(an, bn), big.NewInt(1)
	}
	if op >= 5 {
		tmp := new(big.Rat)
		if op >= 7 {
			tmp.Mul(new(big.Rat).SetInt(bn), c)
		} else {
			tmp.Mul(b, c)
		}
		bn, bd = tmp.Num(), tmp.Denom()
	}
	num := new(big.Int).Mul(an, bd)
	cross := new(big.Int).Mul(bn, ad)
	if op == 1 || op == 6 || op == 8 {
		num.Sub(num, cross)
	} else {
		num.Add(num, cross)
	}
	return num, new(big.Int).Mul(ad, bd)
}

// Exhaust actual free numeric slots so the source post-publication scratch
// replacement must request backing. Retired metadata is not a live numeral.
func addPublicationFillColdCells(t *testing.T, m *RationalManager) []*Rational {
	t.Helper()
	var roots []*Rational
	for m.Z.Cells.Stats().Live+m.Z.Cells.Stats().Retired < m.Z.Cells.Stats().Slots {
		r := NewRational()
		qopsMust(t, m.SetUint64(&r, 4294967297))
		roots = append(roots, &r)
	}
	return roots
}

func TestAdditivePublicationCallerFailurePrefixes(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 9; op++ {
			changed, completed := false, false
			for fault := int64(0); fault < 256; fault++ {
				var m *RationalManager
				var budget *arena.Budget
				if op == 4 {
					m, budget = rationalPublicationDirectOwner(t, sync)
				} else {
					m, budget = rationalManager(t, sync)
				}
				a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
				texts := []string{"1461501637330902918203684832716283019655932542981/1180591620717411303427", "18446744073709551617/4294967297", "4294967297/65537"}
				vals := []*Rational{&a, &b, &c}
				rats := make([]*big.Rat, 3)
				for i, v := range vals {
					qopsMust(t, m.SetString(v, texts[i]))
					rats[i], _ = new(big.Rat).SetString(texts[i])
				}
				qopsMust(t, m.SetInt64(&out, 71))
				pn, pd := addPublicationComponents(op, rats[0], rats[1], rats[2])
				normalized := new(big.Rat).SetFrac(pn, pd)
				fillers := addPublicationFillColdCells(t, m)
				budget.FailAfter(fault)
				e := addPublicationCall(m, op, &a, &b, &c, &out)
				// All caller and retained manager roots are checked with faults armed,
				// before formatting, disabling injection, Del or manager Close.
				addPublicationAudit(t, m, append([]*Rational{&a, &b, &c, &out}, fillers...)...)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal(sync, op, fault, e)
				}
				if e != nil && (out.Num != NewInt(71) || out.Den != NewInt(1)) {
					changed = true
				}
				budget.FailAfter(-1)
				qopsMust(t, m.CleanupAddIntTemporary())
				qopsMust(t, m.CleanupFusedTemporary())
				if e != nil {
					powerPublicationComponent(t, m.Z, &out.Num, "71", pn.String(), normalized.Num().String())
					powerPublicationComponent(t, m.Z, &out.Den, "1", pd.String(), normalized.Denom().String())
				} else {
					completed = true
					got, err := m.String(&out)
					qopsMust(t, err)
					if got != addPublicationModel(op, rats[0], rats[1], rats[2]).RatString() {
						t.Fatal(op, got)
					}
				}
				for i, v := range vals {
					got, err := m.String(v)
					qopsMust(t, err)
					if got != rats[i].RatString() {
						t.Fatal("source input changed", op, i, got)
					}
				}
				for _, v := range append([]*Rational{&a, &b, &c, &out}, fillers...) {
					qopsMust(t, m.Del(v))
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("owner graph leak", op, fault, budget.Used())
				}
				if completed {
					break
				}
			}
			if !completed {
				t.Fatal("finite reservation sweep never completed", sync, op)
			}
			if !changed {
				t.Errorf("source caller publication before a later refusal was hidden: synch=%v op=%d", sync, op)
			}
		}
	}
}
func TestAdditivePublicationFusedUnitDispatch(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for _, unit := range []int64{-1, 1} {
			for op := 5; op < 9; op++ {
				for alias := 0; alias < 4; alias++ {
					a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
					qopsMust(t, m.SetString(&a, "5/7"))
					qopsMust(t, m.SetInt64(&b, unit))
					qopsMust(t, m.SetString(&c, "2/3"))
					dst := []*Rational{&out, &a, &b, &c}[alias]
					qopsMust(t, addPublicationCall(m, op, &a, &b, &c, dst))
					addPublicationAudit(t, m, &a, &b, &c, &out)
					got, e := m.String(dst)
					qopsMust(t, e)
					want := addPublicationModel(op, new(big.Rat).SetFrac64(5, 7), new(big.Rat).SetInt64(unit), new(big.Rat).SetFrac64(2, 3))
					if got != want.RatString() {
						t.Fatal(unit, op, alias, got, want)
					}
					for _, v := range []*Rational{&a, &b, &c, &out} {
						qopsMust(t, m.Del(v))
					}
				}
			}
		}
	}
}

func TestAdditivePublicationIntegerFastpathOldDenominator(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for op := 0; op < 5; op++ {
			changed, completed := false, false
			for fault := int64(0); fault < 128; fault++ {
				m, budget := rationalPublicationDirectOwner(t, sync)
				a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
				qopsMust(t, m.SetString(&a, "1461501637330902918203684832716283019655932542981"))
				qopsMust(t, m.SetString(&b, "18446744073709551617"))
				qopsMust(t, m.SetString(&out, "71/1180591620717411303427"))
				oldDen := out.Den
				fillers := addPublicationFillColdCells(t, m)
				budget.FailAfter(fault)
				e := addPublicationCall(m, op, &a, &b, &c, &out)
				addPublicationAudit(t, m, append([]*Rational{&a, &b, &c, &out}, fillers...)...)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal(op, fault, e)
				}
				if e != nil {
					// reset_denominator is a later source statement, not a deferred rollback.
					if out.Den != oldDen {
						t.Fatal("denominator reset before successful integer action", op, fault, out.Den)
					}
					if out.Num != NewInt(71) {
						changed = true
					}
				} else {
					completed = true
					if out.Den != NewInt(1) {
						t.Fatal("missing source reset_denominator", op, out.Den)
					}
				}
				budget.FailAfter(-1)
				for _, v := range append([]*Rational{&a, &b, &c, &out}, fillers...) {
					qopsMust(t, m.Del(v))
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("fastpath leaked", op, fault)
				}
				if completed {
					break
				}
			}
			if !completed || !changed {
				t.Errorf("integer action publication was hidden op=%d synch=%v completed=%v changed=%v", op, sync, completed, changed)
			}
		}
	}
}

func TestAdditivePublicationRawDenominatorsAndIdentity(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for op := 0; op < 9; op++ {
			for identity := 0; identity < 2; identity++ {
				a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
				// Private construction retains a reachable raw shape instead of SetParts,
				// which deliberately normalizes and changes a negative denominator sign.
				a.Num, a.Den = NewInt(6), NewInt(-8)
				b.Num, b.Den = NewInt(-4), NewInt(10)
				c.Num, c.Den = NewInt(9), NewInt(-12)
				ap, bp, cp := &a, &b, &c
				ar, br, cr := new(big.Rat).SetFrac64(6, -8), new(big.Rat).SetFrac64(-4, 10), new(big.Rat).SetFrac64(9, -12)
				if identity == 1 {
					bp, cp = ap, ap
					br, cr = ar, ar
				}
				qopsMust(t, addPublicationCall(m, op, ap, bp, cp, ap))
				addPublicationAudit(t, m, &a, &b, &c, &out)
				// Mixed methods use raw integer numerator, not normalized oracle.Num().
				rawB := int64(-4)
				if identity == 1 {
					rawB = 6
				}
				want := new(big.Rat)
				switch op {
				case 2:
					want.Add(ar, new(big.Rat).SetInt64(rawB))
				case 3:
					want.Mul(ar, new(big.Rat).SetInt64(rawB))
				case 4:
					want.SetInt64(6 * rawB)
				case 7:
					want.Add(ar, new(big.Rat).Mul(new(big.Rat).SetInt64(rawB), cr))
				case 8:
					want.Sub(ar, new(big.Rat).Mul(new(big.Rat).SetInt64(rawB), cr))
				default:
					want = addPublicationModel(op, ar, br, cr)
				}
				n, e := m.Z.String(&a.Num)
				qopsMust(t, e)
				d, e := m.Z.String(&a.Den)
				qopsMust(t, e)
				ni, nok := new(big.Int).SetString(n, 10)
				di, dok := new(big.Int).SetString(d, 10)
				if !nok || !dok || di.Sign() == 0 {
					t.Fatal("invalid source scalar text", n, d)
				}
				got := new(big.Rat).SetFrac(ni, di)
				if got.Cmp(want) != 0 {
					t.Fatal(sync, op, identity, n, d, want)
				}
				for _, v := range []*Rational{&a, &b, &c, &out} {
					qopsMust(t, m.Del(v))
				}
			}
		}
	}
}

func TestAdditivePublicationMixedScalarComponentAliases(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, _ := rationalManager(t, sync)
		for op := 0; op < 3; op++ {
			for scalarDen := 0; scalarDen < 2; scalarDen++ {
				for sameOut := 0; sameOut < 2; sameOut++ {
					a, out := NewRational(), NewRational()
					a.Num, a.Den = NewInt(5), NewInt(7)
					out.Num, out.Den = NewInt(11), NewInt(13)
					dst := &out
					if sameOut == 1 {
						dst = &a
					}
					scalar := &dst.Num
					k := int64(11)
					if sameOut == 1 {
						k = 5
					}
					if scalarDen == 1 {
						scalar = &dst.Den
						k = 13
						if sameOut == 1 {
							k = 7
						}
					}
					var e error
					want := new(big.Rat)
					switch op {
					case 0:
						e = m.AddInt(&a, scalar, dst)
						want.Add(new(big.Rat).SetFrac64(5, 7), new(big.Rat).SetInt64(k))
					case 1:
						e = m.MulInt(scalar, &a, dst)
						want.Mul(new(big.Rat).SetFrac64(5, 7), new(big.Rat).SetInt64(k))
					case 2:
						e = m.MulInts(&a.Num, scalar, dst)
						want.SetInt64(5 * k)
					}
					qopsMust(t, e)
					addPublicationAudit(t, m, &a, &out)
					got, err := m.String(dst)
					qopsMust(t, err)
					if got != want.RatString() {
						t.Fatal(sync, op, scalarDen, sameOut, got, want)
					}
					qopsMust(t, m.Del(&a))
					qopsMust(t, m.Del(&out))
				}
			}
		}
	}
}

func TestAdditivePublicationSynchronizedTemporaryDeleteOrder(t *testing.T) {
	for _, subtract := range []bool{false, true} {
		actual, _ := rationalManager(t, true)
		reference, _ := rationalManager(t, true)
		aa, ab, ao := NewRational(), NewRational(), NewRational()
		ra, rb, ro := NewRational(), NewRational(), NewRational()
		for _, pair := range []struct {
			m    *RationalManager
			a, b *Rational
		}{{actual, &aa, &ab}, {reference, &ra, &rb}} {
			qopsMust(t, pair.m.SetString(pair.a, "18446744073709551617/4294967297"))
			qopsMust(t, pair.m.SetString(pair.b, "4294967297/65537"))
		}
		// Literal source action transcript, using existing integer algorithms.
		// This is an allocator/lifetime oracle, not replacement rational arithmetic.
		var one, two Int
		qopsMust(t, reference.Z.Mul(&ra.Num, &rb.Den, &one))
		qopsMust(t, reference.Z.Mul(&rb.Num, &ra.Den, &two))
		qopsMust(t, reference.Z.Mul(&ra.Den, &rb.Den, &ro.Den))
		if subtract {
			qopsMust(t, reference.Z.Sub(&one, &two, &ro.Num))
		} else {
			qopsMust(t, reference.Z.Add(&one, &two, &ro.Num))
		}
		qopsMust(t, reference.normalize(&ro))
		qopsMust(t, reference.Z.Del(&one))
		qopsMust(t, reference.Z.Del(&two))
		if subtract {
			qopsMust(t, actual.Sub(&aa, &ab, &ao))
		} else {
			qopsMust(t, actual.Add(&aa, &ab, &ao))
		}
		addPublicationAudit(t, actual, &aa, &ab, &ao)
		addPublicationAudit(t, reference, &ra, &rb, &ro)
		// Cold concrete slab recycling is LIFO. Matching the entire free chain
		// observes both source Del statements, including which cell was last.
		af, rf := actual.Z.Cells.Free, reference.Z.Cells.Free
		for af != 0 || rf != 0 {
			if af != rf {
				t.Fatal("source tmp1 then tmp2 destruction order", subtract, af, rf)
			}
			af = actual.Z.Cells.State(af).Next
			rf = reference.Z.Cells.State(rf).Next
		}
		if rationalString(t, actual, &ao) != rationalString(t, reference, &ro) {
			t.Fatal("source action result")
		}
		for _, v := range []*Rational{&aa, &ab, &ao} {
			qopsMust(t, actual.Del(v))
		}
		for _, v := range []*Rational{&ra, &rb, &ro} {
			qopsMust(t, reference.Del(v))
		}
	}
}
func TestAdditivePublicationWarmPlateau(t *testing.T) {
	for _, sync := range []bool{false, true} {
		m, budget := rationalManager(t, sync)
		a, b, c, out := NewRational(), NewRational(), NewRational(), NewRational()
		qopsMust(t, m.SetString(&a, "18446744073709551617/4294967297"))
		qopsMust(t, m.SetString(&b, "4294967297/65537"))
		qopsMust(t, m.SetString(&c, "65537/257"))
		run := func() {
			for op := 0; op < 9; op++ {
				qopsMust(t, addPublicationCall(m, op, &a, &b, &c, &out))
			}
		}
		for i := 0; i < 100; i++ {
			run()
		}
		before := numericWarmProfileSnapshot(t, m.Z, budget)
		allocs := testing.AllocsPerRun(100, run)
		numericWarmProfileAudit(t, m.Z, budget, before, allocs, SourceExtendedGCDDebug)
		addPublicationAudit(t, m, &a, &b, &c, &out)
		for _, v := range []*Rational{&a, &b, &c, &out} {
			qopsMust(t, m.Del(v))
		}
	}
}

func TestAdditivePublicationServiceValueAndRationalOneRoutes(t *testing.T) {
	s, _ := serviceNew(t)
	a, b := serviceValue(t, s, "5/7"), serviceValue(t, s, "2/3")
	sum, e := s.Add(&a, &b)
	serviceMust(t, e)
	difference, e := s.Sub(&a, &b)
	serviceMust(t, e)
	scalar, e := s.AddInt(&a, -2)
	serviceMust(t, e)
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw, &sum.Raw, &difference.Raw, &scalar.Raw)
	serviceText(t, s, &sum, "29/21")
	serviceText(t, s, &difference, "1/21")
	serviceText(t, s, &scalar, "-9/7")
	// Source copies old value before mutating through rational-one, and copies
	// returned lvalues for arithmetic value results; each result owns its roots.
	old, e := s.PostInc(&a)
	serviceMust(t, e)
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw, &sum.Raw, &difference.Raw, &scalar.Raw, &old.Raw)
	serviceText(t, s, &old, "5/7")
	serviceText(t, s, &a, "12/7")
	serviceMust(t, s.PrefixDec(&a))
	serviceMust(t, s.SetInt64(&a, 99))
	serviceAuditRationalRoots(t, s, &a.Raw, &b.Raw, &sum.Raw, &difference.Raw, &scalar.Raw, &old.Raw)
	serviceText(t, s, &sum, "29/21")
	serviceText(t, s, &difference, "1/21")
	serviceText(t, s, &old, "5/7")
	for _, v := range []*OwnedRational{&a, &b, &sum, &difference, &scalar, &old} {
		serviceMust(t, s.Del(v))
	}
}

// Source normalization divides by the positive GCD without independently
// repairing a denominator's sign. This exact raw-field model is also used by
// the approved dependent failure fixtures; it does not change their math.
func addPublicationExpectRaw(t *testing.T, m *RationalManager, out *Rational, oldNum, oldDen string, pn, pd *big.Int) {
	t.Helper()
	g := new(big.Int).GCD(nil, nil, pn, pd)
	rn, rd := new(big.Int).Quo(new(big.Int).Set(pn), g), new(big.Int).Quo(new(big.Int).Set(pd), g)
	powerPublicationComponent(t, m.Z, &out.Num, oldNum, pn.String(), rn.String())
	powerPublicationComponent(t, m.Z, &out.Den, oldDen, pd.String(), rd.String())
}
