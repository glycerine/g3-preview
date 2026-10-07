// Licensed under the MIT license.
// Source assertions/scenarios: src/test/mpz.cpp tst1, tst2, tst2b.
// Additional finite expectations use independent integer arithmetic.
package util

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func setIntegerOracle(t *testing.T, m *IntegerManager, n *Int, z *big.Int) {
	t.Helper()
	abs := new(big.Int).Abs(new(big.Int).Set(z))
	size := (abs.BitLen() + 31) / 32
	if e := m.SetLimbs(n, wantLimbs(abs, size)); e != nil {
		t.Fatal(e)
	}
	if z.Sign() < 0 {
		if e := m.Neg(n); e != nil {
			t.Fatal(e)
		}
	}
}

func TestIntegerSourceMultiplication(t *testing.T) {
	m, _ := integerManager(t)
	var v, v2, v3 Int
	one := NewInt(1)
	minusTwo := NewInt(-2)
	// tst1 actual arithmetic and comparison assertions. Decimal parsing is
	// independent setup until the source decimal-string setter is ported.
	z, _ := new(big.Int).SetString("1002034040050606089383838288182", 10)
	setIntegerOracle(t, m, &v, z)
	if e := m.Mul(&v, &minusTwo, &v2); e != nil {
		t.Fatal(e)
	}
	if e := m.Add(&v, &v2, &v3); e != nil {
		t.Fatal(e)
	}
	if e := m.Neg(&v3); e != nil {
		t.Fatal(e)
	}
	if cmp, e := m.Compare(&v, &v3); e != nil || cmp != 0 {
		t.Fatal("eq/le/ge and !neq", cmp, e)
	}
	if cmp, e := m.Compare(&v2, &v); e != nil || cmp >= 0 {
		t.Fatal("lt/le/gt/ge and neq", cmp, e)
	}
	// tst2/tst2b are print-only; exact powers are added expectations.
	if e := m.SetInt64(&v, math.MaxUint32); e != nil {
		t.Fatal(e)
	}
	if e := m.Add(&v, &one, &v2); e != nil {
		t.Fatal(e)
	}
	if e := m.Mul(&v2, &v2, &v3); e != nil || integerString(t, m, &v3) != "18446744073709551616" {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		if e := m.Mul(&v2, &v2, &v2); e != nil {
			t.Fatal(e)
		}
	}
	if integerString(t, m, &v2) != new(big.Int).Lsh(big.NewInt(1), 256).String() {
		t.Fatal("source repeated alias square")
	}
}

func TestIntegerSignedDivisionExhaustive(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, q, r, euc, mod Int
	// Finite exact domain, 2352 signed nonzero-divisor cases; direct int64 oracle.
	for x := int64(-24); x <= 24; x++ {
		for y := int64(-24); y <= 24; y++ {
			if y == 0 {
				continue
			}
			m.SetInt64(&a, x)
			m.SetInt64(&b, y)
			if e := m.MachineDivRem(&a, &b, &q, &r); e != nil {
				t.Fatal(e)
			}
			qv, _ := m.Int64(&q)
			rv, _ := m.Int64(&r)
			if qv != x/y || rv != x%y {
				t.Fatal(x, y, qv, rv)
			}
			if e := m.MachineDiv(&a, &b, &q); e != nil {
				t.Fatal(e)
			}
			if e := m.Rem(&a, &b, &r); e != nil {
				t.Fatal(e)
			}
			qv, _ = m.Int64(&q)
			rv, _ = m.Int64(&r)
			if qv != x/y || rv != x%y {
				t.Fatal("single outputs", x, y, qv, rv)
			}
			eq, er := x/y, x%y
			if er < 0 {
				if y > 0 {
					eq--
					er += y
				} else {
					eq++
					er -= y
				}
			}
			if e := m.Div(&a, &b, &euc); e != nil {
				t.Fatal(e)
			}
			if e := m.Mod(&a, &b, &mod); e != nil {
				t.Fatal(e)
			}
			qv, _ = m.Int64(&euc)
			rv, _ = m.Int64(&mod)
			if qv != eq || rv != er {
				t.Fatal("Euclidean", x, y, qv, rv, eq, er)
			}
		}
	}
}

func TestIntegerMulDivisionLimbOracle(t *testing.T) {
	m, _ := integerManager(t)
	rng := rand.New(rand.NewSource(0x514f5452))
	var a, b, out, q, rem Int
	// 500 cases, 1..12 limbs, both signs and distinct/exact alias outputs.
	for trial := 0; trial < 500; trial++ {
		aa, bb := make([]uint32, 1+rng.Intn(12)), make([]uint32, 1+rng.Intn(12))
		for i := range aa {
			aa[i] = rng.Uint32()
		}
		for i := range bb {
			bb[i] = rng.Uint32()
		}
		bb[len(bb)-1] |= 1
		av, bv := oracle(aa), oracle(bb)
		if trial%2 == 0 {
			av.Neg(av)
		}
		if trial%3 == 0 {
			bv.Neg(bv)
		}
		for alias := 0; alias < 3; alias++ {
			setIntegerOracle(t, m, &a, av)
			setIntegerOracle(t, m, &b, bv)
			dst := &out
			if alias == 1 {
				dst = &a
			}
			if alias == 2 {
				dst = &b
			}
			if e := m.Mul(&a, &b, dst); e != nil || integerString(t, m, dst) != new(big.Int).Mul(av, bv).String() {
				t.Fatal("mul", trial, alias, e)
			}
			setIntegerOracle(t, m, &a, av)
			setIntegerOracle(t, m, &b, bv)
			if e := m.MachineDiv(&a, &b, dst); e != nil || integerString(t, m, dst) != new(big.Int).Quo(av, bv).String() {
				t.Fatal("quo", trial, alias, e)
			}
			setIntegerOracle(t, m, &a, av)
			setIntegerOracle(t, m, &b, bv)
			if e := m.Rem(&a, &b, dst); e != nil || integerString(t, m, dst) != new(big.Int).Rem(av, bv).String() {
				t.Fatal("rem", trial, alias, e)
			}
		}
		setIntegerOracle(t, m, &a, av)
		setIntegerOracle(t, m, &b, bv)
		expectedQ, expectedR := new(big.Int), new(big.Int)
		expectedQ.QuoRem(av, bv, expectedR)
		for _, reverse := range []bool{false, true} {
			setIntegerOracle(t, m, &a, av)
			setIntegerOracle(t, m, &b, bv)
			qp, rp := &a, &b
			if reverse {
				qp, rp = &b, &a
			}
			if e := m.MachineDivRem(&a, &b, qp, rp); e != nil || integerString(t, m, qp) != expectedQ.String() || integerString(t, m, rp) != expectedR.String() {
				t.Fatal("pair alias", trial, reverse, e)
			}
		}
		setIntegerOracle(t, m, &a, av)
		setIntegerOracle(t, m, &b, bv)
		expectedQ.DivMod(av, bv, expectedR)
		if e := m.Div(&a, &b, &q); e != nil || integerString(t, m, &q) != expectedQ.String() {
			t.Fatal("euclidean quo", trial, e)
		}
		if e := m.Mod(&a, &b, &rem); e != nil || integerString(t, m, &rem) != expectedR.String() {
			t.Fatal("euclidean mod", trial, e)
		}
	}
}

func TestIntegerDivisionCorrectionFixtures(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, q, r Int
	cases := []struct{ a, b []uint32 }{
		{[]uint32{1, 0, 0x80000000}, []uint32{2, 0, 0x80000000}},
		{[]uint32{1, 0, 1}, []uint32{2, 0, 1}},
		{[]uint32{0, 0, 0x80000000}, []uint32{0xffffffff, 0x80000000}},
		{[]uint32{0xffffffff, 0xffffffff, 0xffffffff}, []uint32{0xffffffff, 0xffffffff}},
		{[]uint32{1}, []uint32{0, 0, 1}},
	}
	for _, tc := range cases {
		m.SetLimbs(&a, tc.a)
		m.SetLimbs(&b, tc.b)
		wq, wr := new(big.Int), new(big.Int)
		wq.QuoRem(oracle(tc.a), oracle(tc.b), wr)
		if e := m.MachineDivRem(&a, &b, &q, &r); e != nil || integerString(t, m, &q) != wq.String() || integerString(t, m, &r) != wr.String() {
			t.Fatal(tc, e)
		}
	}
}

func TestIntegerDivisionZeroAndBudgetFailure(t *testing.T) {
	m, budget := integerManager(t)
	a, b, q, r := NewInt(7), NewInt(0), NewInt(19), NewInt(23)
	for _, call := range []func() error{
		func() error { return m.MachineDivRem(&a, &b, &q, &r) }, func() error { return m.MachineDiv(&a, &b, &q) },
		func() error { return m.Rem(&a, &b, &r) }, func() error { return m.Div(&a, &b, &q) }, func() error { return m.Mod(&a, &b, &r) },
	} {
		if e := call(); !errors.Is(e, ErrDivideByZero) || q.Val != 19 || r.Val != 23 {
			t.Fatal("zero divisor", e)
		}
	}
	m.SetLimbs(&a, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17})
	b = NewInt(7)
	budget.FailAfter(0)
	if e := m.MachineDivRem(&a, &b, &q, &r); !numericPhysicalReservationRefusal(e) || q.Val != 19 || r.Val != 23 {
		t.Fatal("failed division published output", e)
	}
	// Source ensure_tmp frees the insufficient old cell before refusal.
	if m.Tmp[0] != 0 {
		t.Fatal("failed scratch replacement was retained")
	}
	mpzPublicationAudit(t, m, &a, &b, &q, &r)
	// ensure_tmp replacement refusal follows old-free, before output setter.
	if m.Tmp[0] != 0 {
		t.Fatal("failed replacement scratch retained")
	}
	mpzPublicationAudit(t, m, &a, &b, &q, &r)
	budget.FailAfter(-1)
	m.Close()
	fresh, _ := integerManager(t)
	a = NewInt(0)
	q, r = NewInt(19), NewInt(23)
	setIntegerOracle(t, fresh, &a, oracle([]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}))
	if e := fresh.MachineDivRem(&a, &b, &q, &r); e != nil {
		t.Fatal(e)
	}
}

func TestIntegerSourceDivisionAliasDefects(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, both Int
	// Source Div rereads b after overwriting it when c==b. Freeze the actual
	// control-flow result separately from mathematical Euclidean expectations.
	for _, tc := range []struct{ a, b, want int64 }{{-26, 7, -2}, {-26, -7, 2}} {
		m.SetInt64(&a, tc.a)
		m.SetInt64(&b, tc.b)
		if e := m.Div(&a, &b, &b); e != nil {
			t.Fatal(e)
		}
		got, _ := m.Int64(&b)
		if got != tc.want {
			t.Fatal("source quotient alias", got, tc.want)
		}
	}
	for _, divisor := range []int64{7, -7} {
		m.SetInt64(&a, -26)
		m.SetInt64(&b, divisor)
		if e := m.Mod(&a, &b, &b); e != nil || integerString(t, m, &b) != "0" {
			t.Fatal("source modulus aliases divisor", e)
		}
	}
	// Same quotient/remainder destination ends in remainder on the small path,
	// but zero on the source smaller-numerator limb guard (Copy r then Reset q).
	a, b = NewInt(26), NewInt(7)
	if e := m.MachineDivRem(&a, &b, &both, &both); e != nil || integerString(t, m, &both) != "5" {
		t.Fatal("small shared outputs", e)
	}
	m.SetInt64(&a, 1<<40)
	m.SetLimbs(&b, []uint32{0, 0, 256})
	if e := m.MachineDivRem(&a, &b, &both, &both); e != nil || integerString(t, m, &both) != "0" {
		t.Fatal("short numerator shared outputs", e)
	}
	m.SetLimbs(&a, []uint32{5, 0, 256})
	m.SetInt64(&b, 1<<40)
	if e := m.MachineDivRem(&a, &b, &both, &both); e != nil || integerString(t, m, &both) != "5" {
		t.Fatal("general shared outputs", e)
	}
}

func TestIntegerDivisionAllAllocationFailurePoints(t *testing.T) {
	failures, successes := 0, 0
	for after := int64(0); after < 32; after++ {
		m, budget := integerManager(t)
		var a, b Int
		q, r := NewInt(19), NewInt(23)
		aa, bb := make([]uint32, 33), make([]uint32, 13)
		for i := range aa {
			aa[i] = uint32(i*93071 + 13)
		}
		for i := range bb {
			bb[i] = uint32(i*37979 + 5)
		}
		m.SetLimbs(&a, aa)
		m.SetLimbs(&b, bb)
		budget.FailAfter(after)
		err := m.MachineDivRem(&a, &b, &q, &r)
		if err != nil {
			failures++
			if !numericPhysicalReservationRefusal(err) {
				t.Fatal(after, err)
			}
			// mpz.h:156/source cpp503–505: scratch can be missing and
			// caller outputs can already own transferred cells. Count actual
			// named roots before formatting, fault disable, or owner cleanup.
			mpzPublicationAudit(t, m, &a, &b, &q, &r)
			budget.FailAfter(-1)
			m.Close()
			if budget.Used() != 0 {
				t.Fatal("failed source continuation cleanup", after, budget.Used())
			}
			continue
		} else {
			successes++
		}
		budget.FailAfter(-1)
		expectedQ, expectedR := new(big.Int), new(big.Int)
		expectedQ.QuoRem(oracle(aa), oracle(bb), expectedR)
		if integerString(t, m, &q) != expectedQ.String() || integerString(t, m, &r) != expectedR.String() {
			t.Fatal("retry result", after)
		}
		m.Close()
		if budget.Used() != 0 {
			t.Fatal("scratch budget leak", after, budget.Used())
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal("failure sweep did not cross every allocation stage", failures, successes)
	}
}

func TestIntegerMulDivisionWarmAllocations(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, q, r, product Int
	m.SetLimbs(&a, []uint32{0x12345678, 0xfedcba98, 0xabcdef01, 1})
	m.SetLimbs(&b, []uint32{0x87654321, 1})
	if e := m.Mul(&a, &b, &product); e != nil {
		t.Fatal(e)
	}
	if e := m.MachineDivRem(&a, &b, &q, &r); e != nil {
		t.Fatal(e)
	}
	var operationError error
	allocations := testing.AllocsPerRun(100, func() {
		operationError = m.Mul(&a, &b, &product)
		if operationError == nil {
			operationError = m.MachineDivRem(&a, &b, &q, &r)
		}
	})
	if operationError != nil || allocations != 0 {
		t.Fatal("warmed arithmetic allocates", allocations, operationError)
	}
	for _, ref := range m.Natural.divScratch {
		if ref == (arena.VectorRef{}) {
			continue
		}
		capacity, e := m.Words.Cap(ref)
		if e != nil || capacity < 16 {
			t.Fatal("source scratch initial capacity", capacity, e)
		}
	}
}

func TestIntegerEuclideanStagedFailurePointsAndAliases(t *testing.T) {
	// 2 operations x 3 destination shapes x 40 reservation positions. Values
	// require multi-page division and nonzero negative remainder adjustment.
	aa, bb := make([]uint32, 33), make([]uint32, 13)
	for i := range aa {
		aa[i] = uint32(i*93071 + 13)
	}
	for i := range bb {
		bb[i] = uint32(i*37979 + 5)
	}
	av := new(big.Int).Neg(oracle(aa))
	bv := oracle(bb)
	for _, modulus := range []bool{false, true} {
		for alias := 0; alias < 3; alias++ {
			failures, successes := 0, 0
			for after := int64(0); after < 40; after++ {
				m, budget := integerManager(t)
				var a, b Int
				out := NewInt(17)
				setIntegerOracle(t, m, &a, av)
				setIntegerOracle(t, m, &b, bv)
				dst := &out
				if alias == 1 {
					dst = &a
				}
				if alias == 2 {
					dst = &b
				}
				budget.FailAfter(after)
				var err error
				if modulus {
					err = m.Mod(&a, &b, dst)
				} else {
					err = m.Div(&a, &b, dst)
				}
				if err != nil {
					failures++
					if !numericPhysicalReservationRefusal(err) {
						t.Fatal(alias, after, err)
					}
					// mpz.h:156 and mpz.cpp:274 publish completed source prefixes.
					// A missing scratch is not repaired for a fabricated retry.
					mpzPublicationAudit(t, m, &a, &b, &out)
					budget.FailAfter(-1)
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("failed source continuation leaked", budget.Used())
					}
					continue
				} else {
					successes++
				}
				budget.FailAfter(-1)
				expectedQ, expectedR := new(big.Int), new(big.Int)
				expectedQ.DivMod(av, bv, expectedR)
				expected := expectedQ
				if modulus {
					expected = expectedR
				}
				if alias == 2 {
					// Exact source-defect expectation, derived from the wrapper's reread of
					// its overwritten divisor, independently of the port's result.
					expectedQ.QuoRem(av, bv, expectedR)
					if modulus {
						expected = expectedR
						if expected.Sign() < 0 {
							expected.SetInt64(0)
						}
					} else {
						expected = expectedQ
						if expectedR.Sign() != 0 {
							if expected.Sign() < 0 {
								expected.Add(expected, big.NewInt(1))
							} else {
								expected.Sub(expected, big.NewInt(1))
							}
						}
					}
				}
				if integerString(t, m, dst) != expected.String() {
					t.Fatal("staged result", modulus, alias, after)
				}
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("staged owner leaked budget", budget.Used())
				}
			}
			if failures == 0 || successes == 0 {
				t.Fatal("failure sweep missed boundary", modulus, alias, failures, successes)
			}
		}
	}
}

func TestIntegerMultiplicationFailurePointsAndAliases(t *testing.T) {
	aa, bb := make([]uint32, 33), make([]uint32, 13)
	for i := range aa {
		aa[i] = uint32(i*13 + 1)
	}
	for i := range bb {
		bb[i] = uint32(i*17 + 3)
	}
	av, bv := oracle(aa), oracle(bb)
	expected := new(big.Int).Mul(av, bv)
	for alias := 0; alias < 3; alias++ {
		failures, successes := 0, 0
		for after := int64(0); after < 16; after++ {
			m, budget := integerManager(t)
			var a, b Int
			out := NewInt(17)
			setIntegerOracle(t, m, &a, av)
			setIntegerOracle(t, m, &b, bv)
			dst := &out
			if alias == 1 {
				dst = &a
			}
			if alias == 2 {
				dst = &b
			}
			budget.FailAfter(after)
			err := m.Mul(&a, &b, dst)
			if err != nil {
				failures++
				if !numericPhysicalReservationRefusal(err) {
					t.Fatal(alias, after, err)
				}
				// mpz.h:156 and mpz.cpp:274 publish completed source prefixes.
				// A missing scratch is not repaired for a fabricated retry.
				mpzPublicationAudit(t, m, &a, &b, &out)
				budget.FailAfter(-1)
				m.Close()
				if budget.Used() != 0 {
					t.Fatal("failed source continuation leaked", budget.Used())
				}
				continue
			} else {
				successes++
			}
			budget.FailAfter(-1)
			if integerString(t, m, dst) != expected.String() {
				t.Fatal(alias, after)
			}
			m.Close()
			if budget.Used() != 0 {
				t.Fatal(budget.Used())
			}
		}
		if failures == 0 || successes == 0 {
			t.Fatal("fault sweep boundary", alias, failures, successes)
		}
	}
}

func TestIntegerDivisionWidthBoundaries(t *testing.T) {
	m, _ := integerManager(t)
	var a, b, q, r Int
	for _, tc := range []struct{ a, b int64 }{{math.MinInt32, -1}, {math.MinInt32, 1}, {math.MinInt64, -1}, {math.MinInt64, 1}, {math.MaxInt64, -1}, {math.MinInt64, math.MaxInt64}, {math.MaxInt64, math.MinInt64}} {
		m.SetInt64(&a, tc.a)
		m.SetInt64(&b, tc.b)
		expectedQ, expectedR := new(big.Int), new(big.Int)
		expectedQ.QuoRem(big.NewInt(tc.a), big.NewInt(tc.b), expectedR)
		if e := m.MachineDivRem(&a, &b, &q, &r); e != nil || integerString(t, m, &q) != expectedQ.String() || integerString(t, m, &r) != expectedR.String() {
			t.Fatal(tc, e)
		}
	}
	// Small-path positive 2^31 promotion must fail before either output changes.
	m.SetInt64(&a, math.MinInt32)
	b = NewInt(-1)
	m.Reset(&q)
	q.Val = 19
	m.Reset(&r)
	r.Val = 23
	// Force cell exhaustion without relying on already retained free capacity:
	// this separate initialized owner has its slot limit at the six source cells plus two real dead adapter descriptors.
	tight, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 1, SlotLimit: 8}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 1}}})
	if e != nil {
		t.Fatal(e)
	}
	if e = tight.MachineDivRem(&a, &b, &q, &r); !errors.Is(e, arena.ErrSlotLimit) || q.Val != 19 || r.Val != 23 {
		t.Fatal("small quotient promotion publishes before allocation", e)
	}
	tight.Close()
	if e := m.MachineDivRem(&a, &b, &a, &b); e != nil || integerString(t, m, &a) != "2147483648" || integerString(t, m, &b) != "0" {
		t.Fatal("small promotion with input aliases", e)
	}
	m.SetLimbs(&a, []uint32{1, 0, 0x80000000})
	m.SetLimbs(&b, []uint32{2, 0, 0x80000000})
	if e := m.MachineDivRem(&a, &b, &q, &q); e != nil || integerString(t, m, &q) != "39614081257132168796771975169" {
		t.Fatal("same output big remainder", e, integerString(t, m, &q))
	}
}
