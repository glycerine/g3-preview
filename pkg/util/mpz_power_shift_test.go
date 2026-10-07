// Source assertions: src/test/mpz.cpp tst_2k; print-only cases receive labeled
// independent expectations. Fixtures are embedded and never read source at runtime.
package util

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func assertIntegerPowerOracle(t *testing.T, m *IntegerManager, a *Int, want *big.Int) {
	t.Helper()
	if a.IsSmall() {
		if !want.IsInt64() || want.Int64() != int64(a.Val) {
			t.Fatal("inline mismatch", a.Val, want.BitLen())
		}
		return
	}
	if (a.Val < 0) != (want.Sign() < 0) || want.Sign() == 0 {
		t.Fatal("big sign mismatch", a.Val, want.Sign())
	}
	b := new(big.Int).Abs(want).Bytes()
	size := (len(b) + 3) / 4
	if m.Cell(a.Cell).Size != uint32(size) {
		t.Fatal("big size mismatch", m.Cell(a.Cell).Size, size)
	}
	for i := 0; i < size; i++ {
		var word uint32
		for j := 0; j < 4; j++ {
			p := len(b) - 1 - i*4 - j
			if p >= 0 {
				word |= uint32(b[p]) << uint(j*8)
			}
		}
		if m.Digit(a.Cell, uint32(i)) != word {
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
	m, _ := integerManager(t)
	a, out := NewInt(math.MinInt32), NewInt(17)
	// Derived from source signed-int twok=1<<31; not observed via a C++ experiment.
	if e := m.MachineDiv2k(&a, 31, &out); e != nil || integerString(t, m, &out) != "1" {
		t.Fatal("source defect", e)
	}
	if e := m.MachineDiv2k(&a, 31, &a); e != nil || a.Val != 1 {
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
			// Generic enormous bases exercise precedence at bit31; bounded
			// -1 above/below covers the complete MaxUint32 wrap route.
			if p == math.MaxUint32 && (base == 3 || base == -2) {
				continue
			}
			// A source scratch transfer failure need not be retryable. Each
			// independent domain case owns its complete fresh manager graph.
			m, budget := integerManager(t)
			a, out := NewInt(base), NewInt(17)
			// Arm the first real growth for exponentially large generic bases.
			large := p >= 1<<31 && (base == 3 || base == -2)
			if large {
				budget.FailAfter(0)
			}
			e := m.Power(&a, p, &out)
			mpzPublicationAudit(t, m, &a, &out)
			if large {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(base, p, e)
				}
				if out != NewInt(1) {
					t.Fatal("first square refusal before mask31", base, out)
				}
				budget.FailAfter(-1)
				m.Del(&out)
			} else if base == 0 && p == 0 && modularSourceDebug {
				if !errors.Is(e, ErrInvalidPower) || out != NewInt(17) {
					t.Fatal("source assertion before Reset", e, out)
				}
			} else if base == -1 && p >= 1<<31 {
				want := int32(1)
				if p&1 != 0 {
					want = -1
				}
				var typed *PowerDomainError
				if !errors.As(e, &typed) || typed.Exponent != p || !errors.Is(e, ErrNonterminatingPower) || out != NewInt(want) {
					t.Fatal(base, p, out, e)
				}
			} else if base == 2 && p >= 1<<31 {
				if !errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrBounds) && !numericPhysicalReservationRefusal(e) {
					t.Fatal(base, p, e)
				}
				mpzPublicationAudit(t, m, &a, &out)
				if out.Val != 1 || out.Cell != m.FailedNullRef {
					t.Fatal("source base-two allocation prefix", out)
				}
				if x := m.check(&out); !errors.Is(x, arena.ErrInvalidSlot) {
					t.Fatal("failed base-two output read", x)
				}
				// Dispose only the failed caller token before the next
				// independent domain fixture; no scratch repair or retry.
				if x := m.Del(&out); x != nil {
					t.Fatal(x)
				}
			} else {
				if e != nil {
					t.Fatal(base, p, e)
				}
				want := new(big.Int).Exp(big.NewInt(int64(base)), new(big.Int).SetUint64(uint64(p)), nil)
				if base == 0 && p == 0 {
					want.SetInt64(0) // release source fast path Reset, not mathematical 0^0
				}
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
			if base == 2 {
				budget.FailAfter(0)
			}
			e := m.Power(&a, 1<<31, &out)
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den})
			if base == 2 {
				if !numericPhysicalReservationRefusal(e) || out.Num != NewInt(1) || out.Den != NewInt(1) {
					t.Fatal("first real square refusal", base, out, e)
				}
				budget.FailAfter(-1)
				if e := m.CleanupRawArithmetic(); e != nil {
					t.Fatal(e)
				}
			} else {
				want := int32(1)
				if base == 0 {
					want = 0
				}
				var typed *PowerDomainError
				if !errors.As(e, &typed) || !errors.Is(e, ErrNonterminatingPower) || out.Num != NewInt(want) || out.Den != NewInt(1) {
					t.Fatal(base, out, e)
				}
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
					setIntegerOracle(t, m.Z, &integer, n)
					m.Z.SetInt64(&result, 17)
					ratDst, intDst := &out, &result
					if alias == 1 {
						ratDst = &a
						intDst = &integer
					}
					_ = rationalString(t, m, &a)
					call := func() error {
						switch op {
						case 0:
							return m.Z.Power(&integer, 7, intDst)
						case 1:
							return m.Z.Mul2k(&integer, 257, intDst)
						case 2:
							return m.Z.MachineDiv2k(&integer, 33, intDst)
						default:
							return m.Power(&a, 7, ratDst)
						}
					}
					budget.FailAfter(after)
					err := call()
					if err != nil {
						failures++
						if !numericPhysicalReservationRefusal(err) {
							t.Fatal(synch, op, alias, after, err)
						}
						// Source power's intermediate integer setters retain completed prefixes.
						qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den, &integer, &result})
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
						assertIntegerPowerOracle(t, m.Z, intDst, new(big.Int).Exp(n, big.NewInt(7), nil))
					case 1:
						assertIntegerPowerOracle(t, m.Z, intDst, new(big.Int).Lsh(n, 257))
					case 2:
						assertIntegerPowerOracle(t, m.Z, intDst, new(big.Int).Quo(n, new(big.Int).Lsh(big.NewInt(1), 33)))
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
		m, budget := rationalManager(t, synch)
		a, out := NewRational(), NewRational()
		m.SetFraction(&a, (1<<35)+7, 13)
		var result Int
		call := func() error {
			if e := m.Power(&a, 3, &out); e != nil {
				return e
			}
			if e := m.Z.Power(&a.Num, 3, &result); e != nil {
				return e
			}
			if e := m.Z.Mul2k(&a.Num, 97, &result); e != nil {
				return e
			}
			return m.Z.MachineDiv2k(&a.Num, 13, &result)
		}
		for i := 0; i < 3; i++ {
			if e := call(); e != nil {
				t.Fatal(e)
			}
		}
		beforeWarm := numericWarmProfileSnapshot(t, m.Z, budget)
		var err error
		got := testing.AllocsPerRun(20, func() { err = call() })
		if err != nil {
			t.Fatal(synch, got, err)
		}
		numericWarmProfileAudit(t, m.Z, budget, beforeWarm, got, modularSourceDebug)
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den, &result})
		var raw Rational
		// Source copies raw 0/0, publishes out=1, then its first unused
		// square reaches normalize's actual Integer division by gcd(0,0).
		if e := m.Power(&raw, 2, &out); !errors.Is(e, ErrDivideByZero) {
			t.Fatal(synch, e)
		}
		if raw != (Rational{}) || out != (Rational{Num: NewInt(1), Den: NewInt(1)}) {
			t.Fatal("source raw power publication", synch, raw, out)
		}
		qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &out.Num, &out.Den, &result, &raw.Num, &raw.Den})
		m.Close()
		if e := m.Z.Power(&result, 2, &result); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
		if e := m.Z.Mul2k(&result, 2, &result); !errors.Is(e, arena.ErrClosed) {
			t.Fatal(e)
		}
		if _, _, e := m.Z.IsPowerOfTwo(&result); !errors.Is(e, arena.ErrClosed) {
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
	e := m.Mul2k(&a, math.MaxUint32, &out)
	mpzPublicationAudit(t, m, &a, &out)
	if (!errors.Is(e, arena.ErrBudget) && !errors.Is(e, arena.ErrBounds) && !numericPhysicalReservationRefusal(e)) || integerString(t, m, &out) != "9223372036854775807" || integerString(t, m, &a) != "9223372036854775807" {
		t.Fatal(e)
	}
	if e := m.MachineDiv2k(&a, math.MaxUint32, &out); e != nil || integerString(t, m, &out) != "0" {
		t.Fatal(e)
	}
}

func TestPowerDomainAliases(t *testing.T) {
	m, budget := integerManager(t)
	for _, base := range []int32{0, 3, -1} {
		a := NewInt(base)
		p := uint32(0)
		if base == 3 {
			p = 1 << 31
			budget.FailAfter(0)
		}
		if base == -1 {
			p = math.MaxUint32
		}
		e := m.Power(&a, p, &a)
		mpzPublicationAudit(t, m, &a)
		switch base {
		case 0:
			if modularSourceDebug {
				if !errors.Is(e, ErrInvalidPower) || a != NewInt(0) {
					t.Fatal(base, a, e)
				}
			} else if e != nil || a != NewInt(0) {
				t.Fatal(base, a, e)
			}
		case 3:
			if !numericPhysicalReservationRefusal(e) || a != NewInt(1) {
				t.Fatal(base, a, e)
			}
			budget.FailAfter(-1)
		case -1:
			if !errors.Is(e, ErrNonterminatingPower) || a != NewInt(-1) {
				t.Fatal(base, a, e)
			}
		}
		// Each case is an independent call. The source general RAW has
		// no destructor on a body error; audit before explicit owning abort.
		serviceMust(t, m.CleanupPower())
		integerPhysicalAudit(t, m, &a)
	}
	for _, synch := range []bool{false, true} {
		rm, b := rationalManager(t, synch)
		a := NewRational()
		qopsMust(t, rm.SetFraction(&a, -17, 13))
		b.FailAfter(0)
		e := rm.Power(&a, 1<<31, &a)
		qopsAuditLiveOwnership(t, rm, []*Int{&a.Num, &a.Den})
		if !numericPhysicalReservationRefusal(e) || a.Num != NewInt(1) || a.Den != NewInt(1) {
			t.Fatal(synch, a, e)
		}
		b.FailAfter(-1)
	}
}
