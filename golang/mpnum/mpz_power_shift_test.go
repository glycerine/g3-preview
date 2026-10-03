// Copyright (c) 2006 Microsoft Corporation
// Source assertions: src/test/mpz.cpp tst_2k; print-only cases receive labeled
// independent expectations. Fixtures are embedded and never read source at runtime.
package mpnum

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func assertIntegerPowerOracle(t *testing.T, m *IntegerManager, a *Int, want *big.Int) {
	t.Helper()
	if a.IsSmall() {
		if !want.IsInt64() || want.Int64() != int64(a.val) {
			t.Fatal("inline mismatch", a.val, want.BitLen())
		}
		return
	}
	if (a.val < 0) != (want.Sign() < 0) || want.Sign() == 0 {
		t.Fatal("big sign mismatch", a.val, want.Sign())
	}
	b := new(big.Int).Abs(want).Bytes()
	size := (len(b) + 3) / 4
	if m.cell(a.cell).size != uint32(size) {
		t.Fatal("big size mismatch", m.cell(a.cell).size, size)
	}
	for i := 0; i < size; i++ {
		var word uint32
		for j := 0; j < 4; j++ {
			p := len(b) - 1 - i*4 - j
			if p >= 0 {
				word |= uint32(b[p]) << uint(j*8)
			}
		}
		if m.digit(a.cell, uint32(i)) != word {
			t.Fatal("limb mismatch", i)
		}
	}
}
func TestIntegerSourceShiftTable(t *testing.T) {
	fixtures := []struct {
		mul bool
		n   string
		k   uint32
	}{
		{true, "120", 32},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 22},
		{false, "102938484858483282832717616263643648481827437292943727163646457588332211", 22},
		{false, "3", 1},
		{false, "3", 2},
		{false, "3", 0},
		{false, "120", 32},
		{false, "120", 0},
		{false, "81", 2},
		{false, "-3", 1},
		{false, "-3", 2},
		{false, "-3", 0},
		{false, "-102", 4},
		{false, "0", 3},
		{false, "0", 1000},
		{false, "7", 10000},
		{false, "-7", 1000},
		{false, "-7", 2},
		{false, "1029384848584832828327176162636436484", 4},
		{false, "1029384848584832828327176162636436484", 100},
		{false, "102938484858483282832717616263643648481827437292943727163646457588332211", 1},
		{false, "102938484858483282832717616263643648481827437292943727163646457588332211", 0},
		{false, "102938484858483282832717616263643648481827437292943727163646457588332211", 4},
		{false, "102938484858483282832717616263643648481827437292943727163646457588332211", 7},
		{false, "102938484858483282832717616263643648433838737661626264364583983298239291919", 100},
		{false, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 100},
		{false, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 177},
		{false, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 77},
		{false, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 32},
		{false, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 64},
		{false, "-11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 64},
		{false, "-11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 128},
		{false, "-1092983874757371817626399990000000", 100},
		{false, "-109298387475737181762639999000000231", 8},
		{false, "-109298387475737181762639999000000231", 16},
		{false, "-109298387475737181762639999000000231", 17},
		{false, "-109298387475737181762639999000000231", 32},
		{true, "3", 1},
		{true, "3", 2},
		{true, "3", 0},
		{true, "120", 32},
		{true, "120", 0},
		{true, "81", 2},
		{true, "-3", 1},
		{true, "-3", 2},
		{true, "-3", 0},
		{true, "-102", 4},
		{true, "0", 3},
		{true, "0", 1000},
		{true, "7", 10000},
		{true, "-7", 1000000},
		{true, "-7", 2},
		{true, "1029384848584832828327176162636436484", 4},
		{true, "1029384848584832828327176162636436484", 100},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 1},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 0},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 4},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 22},
		{true, "102938484858483282832717616263643648481827437292943727163646457588332211", 7},
		{true, "102938484858483282832717616263643648433838737661626264364583983298239291919", 100},
		{true, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 100},
		{true, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 177},
		{true, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 77},
		{true, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 32},
		{true, "11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 64},
		{true, "-11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 64},
		{true, "-11579208923731619542357098500868790785326998466564056403945758400793872761761638458372762", 128},
		{true, "-1092983874757371817626399990000000", 100},
		{true, "-109298387475737181762639999000000231", 8},
		{true, "-109298387475737181762639999000000231", 16},
		{true, "-109298387475737181762639999000000231", 17},
		{true, "-109298387475737181762639999000000231", 32},
	}
	m, _ := integerManager(t)
	for i, f := range fixtures {
		n, ok := new(big.Int).SetString(f.n, 10)
		if !ok {
			t.Fatal(i)
		}
		for alias := 0; alias < 2; alias++ {
			var a, out, pw, expected Int
			if e := m.SetString(&a, f.n); e != nil {
				t.Fatal(e)
			}
			dst := &out
			if alias == 1 {
				dst = &a
			}
			var err error
			if f.mul {
				err = m.Mul2k(&a, f.k, dst)
			} else {
				err = m.MachineDiv2k(&a, f.k, dst)
			}
			if err != nil {
				t.Fatal(i, alias, err)
			}
			two := NewInt(2)
			if err = m.Power(&two, f.k, &pw); err != nil {
				t.Fatal(i, err)
			}
			// The actual source ENSURE compares against multiply/machine-divide by Power(2,k).
			var original Int
			if e := m.SetString(&original, f.n); e != nil {
				t.Fatal(e)
			}
			if f.mul {
				err = m.Mul(&original, &pw, &expected)
			} else {
				err = m.MachineDiv(&original, &pw, &expected)
			}
			if cmp, e := m.Compare(dst, &expected); err != nil || e != nil || cmp != 0 {
				t.Fatal("source shift coherence", i, alias, cmp, err, e)
			}
			want := new(big.Int)
			if f.mul {
				want.Lsh(n, uint(f.k))
			} else {
				want.Quo(n, new(big.Int).Lsh(big.NewInt(1), uint(f.k)))
			}
			assertIntegerPowerOracle(t, m, dst, want)
			for _, v := range []*Int{&a, &out, &pw, &expected, &original} {
				m.Del(v)
			}
		}
	}
}

func TestIntegerPowerAndShiftIndependent(t *testing.T) {
	m, _ := integerManager(t)
	r := rand.New(rand.NewSource(107))
	for i := 0; i < 200; i++ {
		n := roundRandomInteger(r, 1+r.Intn(7))
		p := uint32(r.Intn(16))
		k := uint32(r.Intn(350))
		for alias := 0; alias < 2; alias++ {
			var a, out Int
			dst := &out
			if alias == 1 {
				dst = &a
			}
			for op := 0; op < 3; op++ {
				setIntegerOracle(t, m, &a, n)
				var err error
				want := new(big.Int)
				switch op {
				case 0:
					err = m.Power(&a, p, dst)
					want.Exp(n, new(big.Int).SetUint64(uint64(p)), nil)
				case 1:
					err = m.Mul2k(&a, k, dst)
					want.Lsh(n, uint(k))
				case 2:
					err = m.MachineDiv2k(&a, k, dst)
					want.Quo(n, new(big.Int).Lsh(big.NewInt(1), uint(k)))
				}
				if err != nil {
					t.Fatal(i, op, alias, err)
				}
				assertIntegerPowerOracle(t, m, dst, want)
			}
			m.Del(&a)
			m.Del(&out)
		}
	}
	// Added exact expectations for source tst_pw2's print-only range and tst_div2k_bug.
	var out Int
	two := NewInt(2)
	for p := uint32(0); p < 128; p++ {
		if e := m.Power(&two, p, &out); e != nil {
			t.Fatal(e)
		}
		assertIntegerPowerOracle(t, m, &out, new(big.Int).Lsh(big.NewInt(1), uint(p)))
		yes, k, e := m.IsPowerOfTwo(&out)
		if e != nil || !yes || k != p {
			t.Fatal(p, yes, k, e)
		}
	}
	m.SetUint64(&out, math.MaxUint32)
	if e := m.MachineDiv2k(&out, 32, &out); e != nil || integerString(t, m, &out) != "0" {
		t.Fatal(e)
	}
	for _, n := range []int64{0, -1, 1, 2, 3, 4, 5, math.MinInt32, math.MaxInt32, math.MaxInt64} {
		m.SetInt64(&out, n)
		yes, k, e := m.IsPowerOfTwo(&out)
		want := n > 0 && (uint64(n)&(uint64(n)-1) == 0)
		if e != nil || yes != want || (yes && new(big.Int).Lsh(big.NewInt(1), uint(k)).Cmp(big.NewInt(n)) != 0) {
			t.Fatal(n, yes, k, e)
		}
	}
}
func TestSourceShiftDefectAndPowerDomains(t *testing.T) {
	m, budget := integerManager(t)
	a, out := NewInt(math.MinInt32), NewInt(17)
	// Derived from source signed-int twok=1<<31; not observed via a C++ experiment.
	if e := m.MachineDiv2k(&a, 31, &out); e != nil || integerString(t, m, &out) != "1" {
		t.Fatal("source defect", e)
	}
	if e := m.MachineDiv2k(&a, 31, &a); e != nil || a.val != 1 {
		t.Fatal("aliased source defect", e)
	}
	// The source big-magnitude branch instead produces the mathematical -1.
	var bigMin Int
	if e := m.SetLimbs(&bigMin, []uint32{1 << 31}); e != nil {
		t.Fatal(e)
	}
	if e := m.Neg(&bigMin); e != nil {
		t.Fatal(e)
	}
	if bigMin.IsSmall() {
		t.Fatal("fixture lost big representation")
	}
	if e := m.MachineDiv2k(&bigMin, 31, &bigMin); e != nil || integerString(t, m, &bigMin) != "-1" {
		t.Fatal("source big branch", e)
	}
	m.Del(&bigMin)
	for _, base := range []int32{0, 1, 2, 3, -1, -2} {
		for _, p := range []uint32{0, 1 << 31, math.MaxUint32} {
			a = NewInt(base)
			m.SetInt64(&out, 17)
			before := budget.Used()
			e := m.Power(&a, p, &out)
			invalid := base == 0 && p == 0
			nonterm := p >= 1<<31 && base != 0 && base != 1 && base != 2
			if invalid || nonterm {
				var typed *PowerDomainError
				if !errors.As(e, &typed) || typed.Exponent != p || integerString(t, m, &out) != "17" || budget.Used() != before {
					t.Fatal(base, p, e)
				}
				if invalid && !errors.Is(e, ErrInvalidPower) {
					t.Fatal(e)
				}
				if nonterm && !errors.Is(e, ErrNonterminatingPower) {
					t.Fatal(e)
				}
			} else if base == 2 && p >= 1<<31 {
				if !errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrBounds) {
					t.Fatal(base, p, e)
				}
				if integerString(t, m, &out) != "17" {
					t.Fatal("oversized power published")
				}
			} else {
				if e != nil {
					t.Fatal(base, p, e)
				}
				want := new(big.Int).Exp(big.NewInt(int64(base)), new(big.Int).SetUint64(uint64(p)), nil)
				assertIntegerPowerOracle(t, m, &out, want)
			}
		}
	}
}
func TestRationalPowerIndependentAndDomains(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, budget := rationalManager(t, synch)
		r := rand.New(rand.NewSource(117))
		for i := 0; i < 100; i++ {
			n, d := roundRandomInteger(r, 1+r.Intn(4)), new(big.Int).Abs(roundRandomInteger(r, 1+r.Intn(3)))
			if d.Sign() == 0 {
				d.SetInt64(1)
			}
			p := uint32(r.Intn(10))
			for alias := 0; alias < 2; alias++ {
				a, out := NewRational(), NewRational()
				setRationalOracle(t, m, &a, n, d)
				dst := &out
				if alias == 1 {
					dst = &a
				}
				if e := m.Power(&a, p, dst); e != nil {
					t.Fatal(i, p, e)
				}
				want := new(big.Rat).SetFrac(new(big.Int).Exp(n, new(big.Int).SetUint64(uint64(p)), nil), new(big.Int).Exp(d, new(big.Int).SetUint64(uint64(p)), nil))
				if rationalString(t, m, dst) != want.RatString() {
					t.Fatal(i, p, alias)
				}
				m.Del(&a)
				m.Del(&out)
			}
		}
		a, out := NewRational(), NewRational()
		if e := m.Power(&a, 0, &out); e != nil || rationalString(t, m, &out) != "1" {
			t.Fatal("source rational zero^zero", e)
		}
		for _, base := range []int64{0, 1, 2, -1} {
			m.SetInt64(&a, base)
			m.SetInt64(&out, 17)
			before := budget.Used()
			e := m.Power(&a, 1<<31, &out)
			var typed *PowerDomainError
			if !errors.As(e, &typed) || !errors.Is(e, ErrNonterminatingPower) || rationalString(t, m, &out) != "17" || budget.Used() != before {
				t.Fatal(base, e)
			}
		}
	}
}

func TestPowerShiftFailureCleanup(t *testing.T) {
	n := new(big.Int).Lsh(big.NewInt(1), 125)
	n.Add(n, big.NewInt(7))
	n.Neg(n)
	d := big.NewInt(13)
	failures, successes := 0, 0
	for _, synch := range []bool{false, true} {
		for op := 0; op < 4; op++ {
			for alias := 0; alias < 2; alias++ {
				for after := int64(0); after < 64; after++ {
					m, budget := rationalManager(t, synch)
					a, out := NewRational(), NewRational()
					setRationalOracle(t, m, &a, n, d)
					m.SetInt64(&out, 17)
					var integer, result Int
					setIntegerOracle(t, m.z, &integer, n)
					m.z.SetInt64(&result, 17)
					ratDst, intDst := &out, &result
					if alias == 1 {
						ratDst = &a
						intDst = &integer
					}
					_ = rationalString(t, m, &a)
					call := func() error {
						switch op {
						case 0:
							return m.z.Power(&integer, 7, intDst)
						case 1:
							return m.z.Mul2k(&integer, 257, intDst)
						case 2:
							return m.z.MachineDiv2k(&integer, 33, intDst)
						default:
							return m.Power(&a, 7, ratDst)
						}
					}
					budget.FailAfter(after)
					err := call()
					if err != nil {
						failures++
						if !errors.Is(err, arena.ErrInjectedFailure) {
							t.Fatal(synch, op, alias, after, err)
						}
						// Source power's intermediate integer setters retain completed prefixes.
						qopsAuditLiveOwnership(t, m, []*Int{&a.num, &a.den, &out.num, &out.den, &integer, &result})
						budget.FailAfter(-1)
						m.Close()
						if budget.Used() != 0 {
							t.Fatal("failed power source roots leaked", budget.Used())
						}
						continue
					} else {
						successes++
					}
					budget.FailAfter(-1)
					switch op {
					case 0:
						assertIntegerPowerOracle(t, m.z, intDst, new(big.Int).Exp(n, big.NewInt(7), nil))
					case 1:
						assertIntegerPowerOracle(t, m.z, intDst, new(big.Int).Lsh(n, 257))
					case 2:
						assertIntegerPowerOracle(t, m.z, intDst, new(big.Int).Quo(n, new(big.Int).Lsh(big.NewInt(1), 33)))
					case 3:
						want := new(big.Rat).SetFrac(new(big.Int).Exp(n, big.NewInt(7), nil), new(big.Int).Exp(d, big.NewInt(7), nil))
						if rationalString(t, m, ratDst) != want.RatString() {
							t.Fatal(op, alias, after)
						}
					}
					m.Close()
					if budget.Used() != 0 {
						t.Fatal("power/shift leak", budget.Used())
					}
				}
			}
		}
	}
	if failures == 0 || successes == 0 {
		t.Fatal("power/shift sweep boundary", failures, successes)
	}
}
func TestPowerShiftWarmStorageAndLifetime(t *testing.T) {
	for _, synch := range []bool{false, true} {
		m, _ := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		m.SetFraction(&a, (1<<35)+7, 13)
		var result Int
		call := func() error {
			if e := m.Power(&a, 3, &out); e != nil {
				return e
			}
			if e := m.z.Power(&a.num, 3, &result); e != nil {
				return e
			}
			if e := m.z.Mul2k(&a.num, 97, &result); e != nil {
				return e
			}
			return m.z.MachineDiv2k(&a.num, 13, &result)
		}
		for i := 0; i < 3; i++ {
			if e := call(); e != nil {
				t.Fatal(e)
			}
		}
		var err error
		if got := testing.AllocsPerRun(20, func() { err = call() }); got != 0 || err != nil {
			t.Fatal(synch, got, err)
		}
		var raw Rational
		if e := m.Power(&raw, 2, &out); !errors.Is(e, ErrUninitializedRational) {
			t.Fatal(e)
		}
		m.Close()
		if e := m.z.Power(&result, 2, &result); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
		if e := m.z.Mul2k(&result, 2, &result); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
		if _, _, e := m.z.IsPowerOfTwo(&result); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
	}
}

func TestShiftUnsignedBoundsAndPromotions(t *testing.T) {
	m, _ := integerManager(t)
	for _, n := range []int64{math.MinInt32, math.MinInt32 + 1, math.MaxInt32, math.MinInt64, math.MaxInt64} {
		for _, k := range []uint32{0, 1, 30, 31, 32, 33, 63, 64, 65} {
			for _, left := range []bool{false, true} {
				var a, out Int
				m.SetInt64(&a, n)
				var e error
				want := new(big.Int)
				if left {
					e = m.Mul2k(&a, k, &out)
					want.Lsh(big.NewInt(n), uint(k))
				} else {
					e = m.MachineDiv2k(&a, k, &out)
					want.Quo(big.NewInt(n), new(big.Int).Lsh(big.NewInt(1), uint(k)))
					if n == math.MinInt32 && k == 31 {
						want.SetInt64(1)
					}
				}
				if e != nil {
					t.Fatal(n, k, left, e)
				}
				assertIntegerPowerOracle(t, m, &out, want)
				m.Del(&a)
				m.Del(&out)
			}
		}
	}
	var a, out Int
	m.SetInt64(&a, math.MaxInt64)
	m.SetInt64(&out, 17)
	if e := m.Mul2k(&a, math.MaxUint32, &out); (!errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrBounds)) || integerString(t, m, &out) != "17" || integerString(t, m, &a) != "9223372036854775807" {
		t.Fatal(e)
	}
	if e := m.MachineDiv2k(&a, math.MaxUint32, &out); e != nil || integerString(t, m, &out) != "0" {
		t.Fatal(e)
	}
}

func TestPowerDomainAliases(t *testing.T) {
	m, budget := integerManager(t)
	for _, fixture := range []struct {
		base     int32
		p        uint32
		sentinel error
	}{{0, 0, ErrInvalidPower}, {3, 1 << 31, ErrNonterminatingPower}, {-1, math.MaxUint32, ErrNonterminatingPower}} {
		a := NewInt(fixture.base)
		before := budget.Used()
		e := m.Power(&a, fixture.p, &a)
		if !errors.Is(e, fixture.sentinel) || a.val != fixture.base || !a.IsSmall() || budget.Used() != before {
			t.Fatal(fixture, e)
		}
	}
	for _, synch := range []bool{false, true} {
		rm, b := rationalManager(t, synch)
		a := NewRational()
		rm.SetFraction(&a, -17, 13)
		_ = rationalString(t, rm, &a) // warm owner-budgeted formatting before method invariant
		before := b.Used()
		e := rm.Power(&a, math.MaxUint32, &a)
		if !errors.Is(e, ErrNonterminatingPower) || rationalString(t, rm, &a) != "-17/13" || b.Used() != before {
			t.Fatal(synch, e)
		}
	}
}
